package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kinchoKayaba/pixicast/backend/db"
	"github.com/kinchoKayaba/pixicast/backend/internal/auth"
	"github.com/kinchoKayaba/pixicast/backend/internal/podcast"
	"github.com/kinchoKayaba/pixicast/backend/internal/twitch"
	"github.com/kinchoKayaba/pixicast/backend/internal/youtube"
)

// ChannelSearchResult は検索結果の1チャンネル
type ChannelSearchResult struct {
	ID              string `json:"id"`
	PlatformID      string `json:"platform_id"`
	ExternalID      string `json:"external_id"`
	Handle          string `json:"handle,omitempty"`
	DisplayName     string `json:"display_name"`
	ThumbnailURL    string `json:"thumbnail_url,omitempty"`
	SubscriberCount int64  `json:"subscriber_count,omitempty"`
	IsSubscribed    bool   `json:"is_subscribed"`
	Source          string `json:"source"`
}

// SearchChannelsResponse は検索レスポンス
type SearchChannelsResponse struct {
	Results      []ChannelSearchResult `json:"results"`
	TotalCount   int                   `json:"total_count"`
	Source       string                `json:"source"`
	QuotaWarning string                `json:"quota_warning,omitempty"`
}

// PopularChannelsResponse は人気チャンネルレスポンス
type PopularChannelsResponse struct {
	Results    []ChannelSearchResult `json:"results"`
	TotalCount int                   `json:"total_count"`
}

// cacheEntry は検索結果キャッシュのエントリ
type cacheEntry struct {
	results   []ChannelSearchResult
	expiresAt time.Time
}

// SearchHandler はチャンネル検索ハンドラ
type SearchHandler struct {
	queries      *db.Queries
	youtube      *youtube.Client
	twitch       *twitch.Client
	podcast      *podcast.Client
	firebaseAuth *auth.FirebaseAuth
	quotaTracker *youtube.QuotaTracker
	cache        map[string]cacheEntry
	cacheMu      sync.RWMutex
}

// NewSearchHandler はハンドラを作成
func NewSearchHandler(
	queries *db.Queries,
	youtubeClient *youtube.Client,
	twitchClient *twitch.Client,
	podcastClient *podcast.Client,
	firebaseAuth *auth.FirebaseAuth,
	quotaTracker *youtube.QuotaTracker,
) *SearchHandler {
	return &SearchHandler{
		queries:      queries,
		youtube:      youtubeClient,
		twitch:       twitchClient,
		podcast:      podcastClient,
		firebaseAuth: firebaseAuth,
		quotaTracker: quotaTracker,
		cache: make(map[string]cacheEntry),
	}
}

// validPlatforms は有効なプラットフォーム一覧
var validPlatforms = map[string]bool{
	"youtube": true,
	"twitch":  true,
	"podcast": true,
	"radiko":  true,
}

// SearchChannels はチャンネル検索API
// GET /v1/channels/search?q={query}&platform={platform}&limit={limit}
func (h *SearchHandler) SearchChannels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証
	userID, err := h.getUserID(r)
	if err != nil {
		log.Printf("SearchChannels: auth failed: %v", err)
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// クエリパラメータ解析
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	platform := strings.TrimSpace(r.URL.Query().Get("platform"))
	limitStr := r.URL.Query().Get("limit")

	limit := int32(20)
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = int32(n)
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 20 {
		limit = 20
	}

	// バリデーション
	if len(query) < 2 {
		respondError(w, http.StatusBadRequest, "query must be at least 2 characters")
		return
	}

	if platform != "" && !validPlatforms[platform] {
		respondError(w, http.StatusBadRequest, "unsupported platform: "+platform)
		return
	}

	// URL検出
	if strings.HasPrefix(query, "http://") || strings.HasPrefix(query, "https://") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "use POST /v1/subscriptions for URL input",
			"hint":  "url_detected",
		})
		return
	}

	// @handle検出: @を除去して検索
	if strings.HasPrefix(query, "@") {
		query = strings.TrimPrefix(query, "@")
	}

	// キャッシュチェック
	cacheKey := platform + ":" + query
	h.cacheMu.RLock()
	if entry, ok := h.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		h.cacheMu.RUnlock()
		results := h.annotateSubscriptions(ctx, entry.results, userID)
		respondJSON(w, http.StatusOK, SearchChannelsResponse{
			Results:    results,
			TotalCount: len(results),
			Source:     "cache",
		})
		return
	}
	h.cacheMu.RUnlock()

	// DB検索
	var dbResults []db.Source
	if platform != "" {
		dbResults, err = h.queries.SearchSourcesByPlatform(ctx, db.SearchSourcesByPlatformParams{
			PlatformID: platform,
			Query:      query,
			MaxResults: limit,
		})
	} else {
		dbResults, err = h.queries.SearchSources(ctx, db.SearchSourcesParams{
			Query:      query,
			MaxResults: limit,
		})
	}
	if err != nil {
		log.Printf("SearchChannels: DB search failed: %v", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// DB結果をレスポンス型に変換
	results := make([]ChannelSearchResult, 0, len(dbResults))
	for _, src := range dbResults {
		results = append(results, sourceToResult(src, "db"))
	}

	responseSource := "db"
	quotaWarning := ""

	// 外部API fallback: DB結果が5件未満の場合
	if len(dbResults) < 5 {
		apiResults, apiQuotaWarning := h.searchExternalAPIs(ctx, query, platform, limit-int32(len(dbResults)))
		if apiQuotaWarning != "" {
			quotaWarning = apiQuotaWarning
		}

		if len(apiResults) > 0 {
			// 重複除去: (platform_id, external_id) ベース
			seen := make(map[string]bool)
			for _, r := range results {
				seen[r.PlatformID+":"+r.ExternalID] = true
			}
			for _, r := range apiResults {
				key := r.PlatformID + ":" + r.ExternalID
				if !seen[key] {
					results = append(results, r)
					seen[key] = true
				}
			}
			if len(dbResults) > 0 {
				responseSource = "db+api"
			} else {
				responseSource = "api"
			}
		}
	}

	// 登録者数の多い順にソート（Stable: 同数の場合はAPI関連度順を保持）
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].SubscriberCount > results[j].SubscriberCount
	})

	// limit適用
	if int32(len(results)) > limit {
		results = results[:limit]
	}

	// キャッシュ保存
	h.cacheMu.Lock()
	h.cache[cacheKey] = cacheEntry{
		results:   results,
		expiresAt: time.Now().Add(30 * time.Minute),
	}
	h.cacheMu.Unlock()

	// is_subscribedアノテーション
	results = h.annotateSubscriptions(ctx, results, userID)

	respondJSON(w, http.StatusOK, SearchChannelsResponse{
		Results:      results,
		TotalCount:   len(results),
		Source:       responseSource,
		QuotaWarning: quotaWarning,
	})
}

// PopularChannels は人気チャンネルAPI
// GET /v1/channels/popular?limit={limit}
func (h *SearchHandler) PopularChannels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := h.getUserID(r)
	if err != nil {
		log.Printf("PopularChannels: auth failed: %v", err)
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := int32(10)
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = int32(n)
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 20 {
		limit = 20
	}

	rows, err := h.queries.PopularSources(ctx, limit)
	if err != nil {
		log.Printf("PopularChannels: DB query failed: %v", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	results := make([]ChannelSearchResult, 0, len(rows))
	for _, row := range rows {
		result := ChannelSearchResult{
			ID:              row.ID.String(),
			PlatformID:      row.PlatformID,
			ExternalID:      row.ExternalID,
			DisplayName:     pgTextValue(row.DisplayName),
			ThumbnailURL:    pgTextValue(row.ThumbnailUrl),
			SubscriberCount: row.SubscriberCount,
			Source:          "db",
		}
		if row.Handle.Valid {
			result.Handle = row.Handle.String
		}
		results = append(results, result)
	}

	// is_subscribedアノテーション
	results = h.annotateSubscriptions(ctx, results, userID)

	respondJSON(w, http.StatusOK, PopularChannelsResponse{
		Results:    results,
		TotalCount: len(results),
	})
}

// searchExternalAPIs は外部APIフォールバック検索
func (h *SearchHandler) searchExternalAPIs(ctx context.Context, query string, platform string, maxResults int32) ([]ChannelSearchResult, string) {
	var results []ChannelSearchResult
	quotaWarning := ""

	if maxResults <= 0 {
		maxResults = 5
	}

	// YouTube検索
	if platform == "" || platform == "youtube" {
		if h.quotaTracker.GetUsagePercent() >= 80 {
			quotaWarning = "youtube_quota_limited"
			log.Printf("SearchChannels: YouTube quota limited (%.1f%%), skipping external search", h.quotaTracker.GetUsagePercent())
		} else if h.quotaTracker.CanUse(101) {
			ytResults, err := h.youtube.SearchChannels(ctx, query, int64(maxResults))
			if err != nil {
				log.Printf("SearchChannels: YouTube API search failed: %v", err)
			} else {
				h.quotaTracker.RecordUsage(ctx, "search.list", 100)
				h.quotaTracker.RecordUsage(ctx, "channels.list", 1)

				for _, yt := range ytResults {
					ch := ChannelSearchResult{
						PlatformID:      "youtube",
						ExternalID:      yt.ChannelID,
						Handle:          yt.Handle,
						DisplayName:     yt.DisplayName,
						ThumbnailURL:    yt.ThumbnailURL,
						SubscriberCount: yt.SubscriberCount,
						Source:          "api",
					}
					results = append(results, ch)
					h.upsertChannelFromAPI(ctx, "youtube", ch)
				}
			}
		}
	}

	// Twitch検索
	if platform == "" || platform == "twitch" {
		twResults, err := h.twitch.SearchChannels(ctx, query, int(maxResults))
		if err != nil {
			log.Printf("SearchChannels: Twitch API search failed: %v", err)
		} else {
			for _, tw := range twResults {
				ch := ChannelSearchResult{
					PlatformID:   "twitch",
					ExternalID:   tw.ID,
					Handle:       tw.BroadcasterLogin,
					DisplayName:  tw.DisplayName,
					ThumbnailURL: tw.ThumbnailURL,
					Source:       "api",
				}
				results = append(results, ch)
				h.upsertChannelFromAPI(ctx, "twitch", ch)
			}
		}
	}

	// Podcast検索
	if platform == "" || platform == "podcast" {
		podResults, err := h.podcast.SearchPodcasts(ctx, query, int(maxResults))
		if err != nil {
			log.Printf("SearchChannels: Podcast API search failed: %v", err)
		} else {
			for _, pod := range podResults {
				externalID := fmt.Sprintf("id%d", pod.CollectionID)
				ch := ChannelSearchResult{
					PlatformID:   "podcast",
					ExternalID:   externalID,
					DisplayName:  pod.TrackName,
					ThumbnailURL: pod.ArtworkURL,
					Source:       "api",
				}
				results = append(results, ch)
				h.upsertChannelFromAPI(ctx, "podcast", ch)
			}
		}
	}

	// Radiko: DB-onlyのためAPI検索なし

	return results, quotaWarning
}

// upsertChannelFromAPI は外部API結果をDBにupsert
func (h *SearchHandler) upsertChannelFromAPI(ctx context.Context, platform string, ch ChannelSearchResult) {
	_, err := h.queries.UpsertSource(ctx, db.UpsertSourceParams{
		PlatformID:   platform,
		ExternalID:   ch.ExternalID,
		Handle:       toPgText(ch.Handle),
		DisplayName:  toPgText(ch.DisplayName),
		ThumbnailUrl: toPgText(ch.ThumbnailURL),
	})
	if err != nil {
		log.Printf("SearchChannels: failed to upsert source %s/%s: %v", platform, ch.ExternalID, err)
	}
}

// annotateSubscriptions は結果にis_subscribedフラグを付与
func (h *SearchHandler) annotateSubscriptions(ctx context.Context, results []ChannelSearchResult, userID int64) []ChannelSearchResult {
	if userID == 0 || len(results) == 0 {
		return results
	}

	subs, err := h.queries.ListUserEnabledSubscriptions(ctx, userID)
	if err != nil {
		log.Printf("SearchChannels: failed to get user subscriptions: %v", err)
		return results
	}

	subscribedSet := make(map[string]bool)
	for _, sub := range subs {
		subscribedSet[sub.PlatformID+":"+sub.ExternalID] = true
	}

	for i := range results {
		key := results[i].PlatformID + ":" + results[i].ExternalID
		results[i].IsSubscribed = subscribedSet[key]
	}

	return results
}

// getUserID はリクエストからuser_idを取得
func (h *SearchHandler) getUserID(r *http.Request) (int64, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return 0, fmt.Errorf("authorization header is required")
	}

	idToken, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		return 0, err
	}

	token, err := h.firebaseAuth.VerifyIDToken(r.Context(), idToken)
	if err != nil {
		return 0, fmt.Errorf("failed to verify token: %w", err)
	}

	return auth.GetUserIDFromToken(token), nil
}

// sourceToResult はdb.SourceをChannelSearchResultに変換
func sourceToResult(src db.Source, source string) ChannelSearchResult {
	r := ChannelSearchResult{
		ID:         src.ID.String(),
		PlatformID: src.PlatformID,
		ExternalID: src.ExternalID,
		Source:     source,
	}
	if src.Handle.Valid {
		r.Handle = src.Handle.String
	}
	if src.DisplayName.Valid {
		r.DisplayName = src.DisplayName.String
	}
	if src.ThumbnailUrl.Valid {
		r.ThumbnailURL = src.ThumbnailUrl.String
	}
	return r
}

// pgTextValue はpgtype.Textから文字列を取得
func pgTextValue(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

// toPgText は文字列からpgtype.Textを作成
func toPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// respondJSON はJSONレスポンスを返す
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
