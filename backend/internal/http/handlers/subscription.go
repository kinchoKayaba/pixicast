package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kinchoKayaba/pixicast/backend/db"
	"github.com/kinchoKayaba/pixicast/backend/internal/auth"
	"github.com/kinchoKayaba/pixicast/backend/internal/ingest"
	"github.com/kinchoKayaba/pixicast/backend/internal/podcast"
	"github.com/kinchoKayaba/pixicast/backend/internal/twitch"
	"github.com/kinchoKayaba/pixicast/backend/internal/youtube"
)

// SubscriptionHandler は購読関連のハンドラ
type SubscriptionHandler struct {
	queries      *db.Queries
	youtube      *youtube.Client
	twitch       *twitch.Client
	podcast      *podcast.Client
	firebaseAuth *auth.FirebaseAuth
}

// NewSubscriptionHandler はハンドラを作成
func NewSubscriptionHandler(queries *db.Queries, youtubeClient *youtube.Client, twitchClient *twitch.Client, podcastClient *podcast.Client, firebaseAuth *auth.FirebaseAuth) *SubscriptionHandler {
	return &SubscriptionHandler{
		queries:      queries,
		youtube:      youtubeClient,
		twitch:       twitchClient,
		podcast:      podcastClient,
		firebaseAuth: firebaseAuth,
	}
}

// CreateSubscriptionRequest はリクエストJSON
type CreateSubscriptionRequest struct {
	Platform string `json:"platform"` // "youtube"
	Input    string `json:"input"`    // URL or @handle or UCxxx...
}

// CreateSubscriptionResponse はレスポンスJSON
type CreateSubscriptionResponse struct {
	Subscription SubscriptionData `json:"subscription"`
}

// SubscriptionData は購読情報
type SubscriptionData struct {
	UserID       int64  `json:"user_id"`
	Platform     string `json:"platform"`
	SourceID     string `json:"source_id"`
	ChannelID    string `json:"channel_id"`
	Handle       string `json:"handle,omitempty"`
	DisplayName  string `json:"display_name"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Enabled      bool   `json:"enabled"`
	IsFavorite   bool   `json:"is_favorite"`
}

// ErrorResponse はエラーレスポンス
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateSubscription は購読登録API
// POST /v1/subscriptions
func (h *SubscriptionHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Firebase認証: user_idを取得
	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		log.Printf("Authentication failed: %v", err)
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// プラン別チャンネル数制限チェック
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		idToken, err := auth.ExtractTokenFromHeader(authHeader)
		if err == nil && idToken != "" {
			token, err := h.firebaseAuth.VerifyIDToken(ctx, idToken)
			if err == nil && token != nil {
				planType := auth.GetPlanTypeFromToken(token)
				log.Printf("📊 Plan check - planType: %s, userID: %d", planType, userID)
				
				planLimit, err := h.queries.GetPlanLimit(ctx, planType)
				if err != nil {
					log.Printf("❌ Failed to get plan limit for %s: %v", planType, err)
					planLimit.MaxChannels = 5
				}
				log.Printf("📊 Plan limit - max_channels: %d, display_name: %s", planLimit.MaxChannels, planLimit.DisplayName)
				
				count, err := h.queries.CountUserSubscriptions(ctx, userID)
				if err != nil {
					log.Printf("❌ Failed to count subscriptions: %v", err)
				} else {
					log.Printf("📊 Current subscriptions: %d / %d", count, planLimit.MaxChannels)
					if count >= int64(planLimit.MaxChannels) {
						log.Printf("🚫 Subscription limit reached for planType: %s", planType)
						if planType == "free_anonymous" {
							respondError(w, http.StatusForbidden, fmt.Sprintf("匿名ユーザーは%dチャンネルまでしか登録できません。Googleログインして無制限に登録しましょう！", planLimit.MaxChannels))
						} else {
							respondError(w, http.StatusForbidden, fmt.Sprintf("%sプランは%dチャンネルまでです。", planLimit.DisplayName, planLimit.MaxChannels))
						}
						return
					}
				}
			}
		}
	}

	// リクエスト解析
	var req CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// バリデーション
	if req.Platform != "youtube" && req.Platform != "twitch" && req.Platform != "podcast" {
		respondError(w, http.StatusBadRequest, "only youtube, twitch, and podcast platforms are supported")
		return
	}
	if req.Input == "" {
		respondError(w, http.StatusBadRequest, "input is required")
		return
	}

	// プラットフォーム別処理
	switch req.Platform {
	case "youtube":
		h.handleYouTubeSubscription(ctx, w, req, userID)
	case "twitch":
		h.handleTwitchSubscription(ctx, w, req, userID)
	case "podcast":
		h.handlePodcastSubscription(ctx, w, req, userID)
	default:
		respondError(w, http.StatusBadRequest, fmt.Sprintf("unsupported platform: %s", req.Platform))
	}
}

// getUserIDFromRequest はリクエストからuser_idを取得
func (h *SubscriptionHandler) getUserIDFromRequest(r *http.Request) (int64, error) {
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

	userID := auth.GetUserIDFromToken(token)
	log.Printf("✅ Authenticated user: Firebase UID=%s, user_id=%d, anonymous=%v", token.UID, userID, token.Firebase.SignInProvider == "anonymous")
	return userID, nil
}

// handleYouTubeSubscription はYouTube購読処理
func (h *SubscriptionHandler) handleYouTubeSubscription(ctx context.Context, w http.ResponseWriter, req CreateSubscriptionRequest, userID int64) {
	channelID, handle, err := h.normalizeInput(req.Input)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("Normalized YouTube input: channelID=%s, handle=%s", channelID, handle)

	if channelID == "" && handle != "" {
		resolvedID, err := h.youtube.ResolveHandle(ctx, handle)
		if err != nil {
			log.Printf("Failed to resolve handle @%s: %v", handle, err)
			respondError(w, http.StatusNotFound, fmt.Sprintf("channel not found for handle: @%s", handle))
			return
		}
		channelID = resolvedID
		log.Printf("Resolved @%s to channelID: %s", handle, channelID)
	}

	details, err := h.youtube.GetChannelDetails(ctx, channelID)
	if err != nil {
		log.Printf("Failed to get channel details for %s: %v", channelID, err)
		respondError(w, http.StatusNotFound, "channel not found")
		return
	}

	if handle == "" && details.Handle != "" {
		handle = details.Handle
	}

	log.Printf("YouTube channel details: %+v", details)

	subscription, err := h.upsertYouTubeSubscription(ctx, userID, req.Platform, details)
	if err != nil {
		log.Printf("Failed to upsert YouTube subscription: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateSubscriptionResponse{
		Subscription: *subscription,
	})
}

// normalizeInput は入力を正規化してchannelIDまたはhandleを抽出
// 戻り値: (channelID, handle, error)
func (h *SubscriptionHandler) normalizeInput(input string) (string, string, error) {
	input = strings.TrimSpace(input)

	// URLの場合
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return h.parseYouTubeURL(input)
	}

	// @handle の場合
	if strings.HasPrefix(input, "@") {
		handle := strings.TrimPrefix(input, "@")
		if handle == "" {
			return "", "", fmt.Errorf("invalid handle")
		}
		return "", handle, nil
	}

	// UCxxx... の場合（channelID）
	if strings.HasPrefix(input, "UC") {
		return input, "", nil
	}

	return "", "", fmt.Errorf("invalid input format")
}

// parseYouTubeURL はYouTube URLをパースしてchannelIDまたはhandleを抽出
func (h *SubscriptionHandler) parseYouTubeURL(rawURL string) (string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL")
	}

	// youtube.com または youtu.be のみ許可
	if u.Host != "www.youtube.com" && u.Host != "youtube.com" && u.Host != "youtu.be" {
		return "", "", fmt.Errorf("not a YouTube URL")
	}

	path := strings.TrimPrefix(u.Path, "/")

	// /channel/UCxxx... の形式
	if strings.HasPrefix(path, "channel/") {
		channelID := strings.TrimPrefix(path, "channel/")
		// パスの最初のセグメントのみ取得（/featured等を除去）
		parts := strings.Split(channelID, "/")
		channelID = parts[0]
		if strings.HasPrefix(channelID, "UC") {
			return channelID, "", nil
		}
	}

	// /@handle の形式
	if strings.HasPrefix(path, "@") {
		handle := path
		// パスの最初のセグメントのみ取得（/featured等を除去）
		parts := strings.Split(handle, "/")
		handle = strings.TrimPrefix(parts[0], "@")
		if handle != "" {
			return "", handle, nil
		}
	}

	return "", "", fmt.Errorf("could not extract channel ID or handle from URL")
}

// subscriptionResult は内部用の購読結果
type subscriptionResult struct {
	UserID   int64
	SourceID string
	Enabled  bool
}

// upsertSubscription はsourcesとuser_subscriptionsをupsert
func (h *SubscriptionHandler) upsertYouTubeSubscription(
	ctx context.Context,
	userID int64,
	platform string,
	details *youtube.ChannelDetails,
) (*SubscriptionData, error) {
	// sourcesをupsert
	source, err := h.queries.UpsertSource(ctx, db.UpsertSourceParams{
		PlatformID: platform,
		ExternalID: details.ChannelID,
		Handle: pgtype.Text{
			String: details.Handle,
			Valid:  details.Handle != "",
		},
		DisplayName: pgtype.Text{
			String: details.DisplayName,
			Valid:  details.DisplayName != "",
		},
		ThumbnailUrl: pgtype.Text{
			String: details.ThumbnailURL,
			Valid:  details.ThumbnailURL != "",
		},
		UploadsPlaylistID: pgtype.Text{
			String: details.UploadsPlaylistID,
			Valid:  details.UploadsPlaylistID != "",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert source: %w", err)
	}

	log.Printf("Upserted source: %s (id=%s)", source.ExternalID, source.ID.String())

	// user_subscriptionsをupsert
	subscription, err := h.queries.UpsertUserSubscription(ctx, db.UpsertUserSubscriptionParams{
		UserID:   userID,
		SourceID: source.ID,
		Enabled:  true,
		Priority: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user subscription: %w", err)
	}

	log.Printf("Upserted user_subscription: user_id=%d, source_id=%s", userID, source.ID.String())

	// チャンネル追加後、2025/1/1以降の全動画を取得してDBに保存
	go func() {
		ingestCtx := context.Background()
		since := "2025-01-01T00:00:00Z"
		if err := ingest.FetchAndSaveChannelVideosSince(
			ingestCtx,
			h.queries,
			h.youtube,
			source.ID,
			details.ChannelID,
			0, // 全動画
			since,
		); err != nil {
			log.Printf("Failed to fetch videos for channel %s: %v", details.ChannelID, err)
		}
	}()

	return &SubscriptionData{
		UserID:       userID,
		Platform:     platform,
		SourceID:     source.ID.String(),
		ChannelID:    details.ChannelID,
		Handle:       details.Handle,
		DisplayName:  details.DisplayName,
		ThumbnailURL: details.ThumbnailURL,
		Enabled:      subscription.Enabled,
	}, nil
}

// enqueueIngest は非同期取り込みキック（スタブ）
func (h *SubscriptionHandler) enqueueIngest(sourceID string) {
	// TODO: 将来はCloud Tasks / PubSubに差し替え
	log.Printf("TODO: Enqueue ingest for source_id=%s", sourceID)
	
	// 実際の取り込み処理はここに実装予定
	// 例：
	// - sourcesからuploads_playlist_idを取得
	// - PlaylistItems APIで動画一覧を取得
	// - programsテーブルに保存
}

// ListSubscriptions は購読一覧取得API
// GET /v1/subscriptions
func (h *SubscriptionHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証: user_idを取得（認証なしの場合は空リスト返却）
	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"subscriptions": []SubscriptionData{}})
		return
	}

	// 購読一覧を取得
	subscriptions, err := h.queries.ListUserEnabledSubscriptions(ctx, userID)
	if err != nil {
		log.Printf("Failed to list subscriptions: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch subscriptions")
		return
	}

	// レスポンス用に変換
	var responseData []SubscriptionData
	for _, sub := range subscriptions {
		// NULL許容フィールドの処理
		handle := ""
		if sub.Handle.Valid {
			handle = sub.Handle.String
		}
		displayName := ""
		if sub.DisplayName.Valid {
			displayName = sub.DisplayName.String
		}
		thumbnailURL := ""
		if sub.ThumbnailUrl.Valid {
			thumbnailURL = sub.ThumbnailUrl.String
		}

		responseData = append(responseData, SubscriptionData{
			UserID:       userID,
			Platform:     sub.PlatformID,
			SourceID:     sub.ID.String(),
			ChannelID:    sub.ExternalID,
			Handle:       handle,
			DisplayName:  displayName,
			ThumbnailURL: thumbnailURL,
			Enabled:      sub.Enabled,
			IsFavorite:   sub.IsFavorite,
		})
	}

	// レスポンス
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subscriptions": responseData,
	})
}

// DeleteSubscription はチャンネル登録を解除
func (h *SubscriptionHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	log.Printf("DeleteSubscription called: method=%s, path=%s", r.Method, r.URL.Path)

	// CORSヘッダー（最初に設定）
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodDelete {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 認証: user_idを取得
	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		log.Printf("Authentication failed: %v", err)
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// URLからチャンネルIDを取得 (/v1/subscriptions/{channelId})
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/v1/subscriptions/"), "/")
	log.Printf("Path parts: %v", parts)
	
	if len(parts) == 0 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "Channel ID is required")
		return
	}
	channelID := parts[0]
	log.Printf("Deleting channel: %s", channelID)

	ctx := context.Background()

	// チャンネルを検索（複数のプラットフォームを試す）
	var source db.Source
	var findErr error
	
	// YouTube, Twitch, Podcastの順に試す
	platforms := []string{"youtube", "twitch", "podcast"}
	found := false
	
	for _, platform := range platforms {
		source, findErr = h.queries.GetSourceByExternalID(ctx, db.GetSourceByExternalIDParams{
			PlatformID: platform,
			ExternalID: channelID,
		})
		if findErr == nil {
			log.Printf("✅ Found source on platform: %s", platform)
			found = true
			break
		}
	}
	
	if !found {
		log.Printf("❌ Failed to find source with ID %s on any platform", channelID)
		respondError(w, http.StatusNotFound, "Channel not found")
		return
	}

	// 購読を削除（enabledをfalseにする）
	err = h.queries.DeleteUserSubscription(ctx, db.DeleteUserSubscriptionParams{
		UserID:   userID,
		SourceID: source.ID,
	})
	if err != nil {
		log.Printf("Failed to delete subscription: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to delete subscription")
		return
	}

	// 成功レスポンス
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Subscription deleted successfully",
	})
}

// handleTwitchSubscription はTwitch購読処理
func (h *SubscriptionHandler) handleTwitchSubscription(ctx context.Context, w http.ResponseWriter, req CreateSubscriptionRequest, userID int64) {
	login := strings.TrimPrefix(strings.TrimPrefix(req.Input, "https://www.twitch.tv/"), "@")
	user, err := h.twitch.GetUserByLogin(ctx, login)
	if err != nil {
		log.Printf("Failed to get Twitch user: %v", err)
		respondError(w, http.StatusNotFound, "Twitch user not found")
		return
	}

	source, err := h.queries.UpsertSource(ctx, db.UpsertSourceParams{
		PlatformID: "twitch", ExternalID: user.ID,
		Handle: pgtype.Text{String: user.Login, Valid: true},
		DisplayName: pgtype.Text{String: user.DisplayName, Valid: true},
		ThumbnailUrl: pgtype.Text{String: user.ProfileImageURL, Valid: true},
		UploadsPlaylistID: pgtype.Text{},
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}

	subscription, err := h.queries.UpsertUserSubscription(ctx, db.UpsertUserSubscriptionParams{
		UserID: userID, SourceID: source.ID, Enabled: true, Priority: 0,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}

	go func() {
		since := "2025-01-01T00:00:00Z"
		ingest.FetchAndSaveTwitchVideosSince(context.Background(), h.queries, h.twitch, source.ID, user.ID, since)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateSubscriptionResponse{
		Subscription: SubscriptionData{
			UserID: userID, Platform: "twitch", SourceID: source.ID.String(),
			ChannelID: user.ID, Handle: user.Login, DisplayName: user.DisplayName,
			ThumbnailURL: user.ProfileImageURL, Enabled: subscription.Enabled,
		},
	})
}

// handlePodcastSubscription はPodcast購読処理
func (h *SubscriptionHandler) handlePodcastSubscription(ctx context.Context, w http.ResponseWriter, req CreateSubscriptionRequest, userID int64) {
	// Apple PodcastsのURLからRSSフィードURLを取得
	feedURL, err := h.podcast.ResolveFeedURL(ctx, req.Input)
	if err != nil {
		log.Printf("Failed to resolve feed URL: %v", err)
		respondError(w, http.StatusBadRequest, "failed to resolve podcast feed URL")
		return
	}
	log.Printf("Resolved feed URL: %s", feedURL)

	podcastFeed, _, err := h.podcast.ParseFeed(ctx, feedURL)
	if err != nil {
		log.Printf("Failed to parse podcast feed: %v", err)
		respondError(w, http.StatusBadRequest, "invalid podcast feed URL")
		return
	}

	source, err := h.queries.UpsertSource(ctx, db.UpsertSourceParams{
		PlatformID: "podcast", ExternalID: feedURL,
		Handle: pgtype.Text{},
		DisplayName: pgtype.Text{String: podcastFeed.Title, Valid: true},
		ThumbnailUrl: pgtype.Text{String: podcastFeed.ImageURL, Valid: true},
		UploadsPlaylistID: pgtype.Text{},
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}

	subscription, err := h.queries.UpsertUserSubscription(ctx, db.UpsertUserSubscriptionParams{
		UserID: userID, SourceID: source.ID, Enabled: true, Priority: 0,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}

	go func() {
		since := "2025-01-01T00:00:00Z"
		ingest.FetchAndSavePodcastEpisodesSince(context.Background(), h.queries, h.podcast, source.ID, feedURL, since)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateSubscriptionResponse{
		Subscription: SubscriptionData{
			UserID: userID, Platform: "podcast", SourceID: source.ID.String(),
			ChannelID: feedURL, Handle: "", DisplayName: podcastFeed.Title,
			ThumbnailURL: podcastFeed.ImageURL, Enabled: subscription.Enabled,
		},
	})
}

// ToggleFavorite はお気に入り状態を切り替え
func (h *SubscriptionHandler) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// プランチェック: お気に入りはBasic以上
	authHeader := r.Header.Get("Authorization")
	idToken, _ := auth.ExtractTokenFromHeader(authHeader)
	token, _ := h.firebaseAuth.VerifyIDToken(r.Context(), idToken)
	if token != nil && auth.GetPlanTypeFromToken(token) == "free_anonymous" {
		respondError(w, http.StatusForbidden, "お気に入り機能はGoogleログイン後に利用できます！")
		return
	}

	var reqBody struct {
		IsFavorite bool `json:"is_favorite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/v1/subscriptions/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "Channel ID is required")
		return
	}
	channelID := parts[0]

	ctx := context.Background()
	
	// チャンネルを検索（複数のプラットフォームを試す）
	var source db.Source
	var findErr error
	platforms := []string{"youtube", "twitch", "podcast"}
	found := false
	
	for _, platform := range platforms {
		source, findErr = h.queries.GetSourceByExternalID(ctx, db.GetSourceByExternalIDParams{
			PlatformID: platform,
			ExternalID: channelID,
		})
		if findErr == nil {
			found = true
			break
		}
	}
	
	if !found {
		respondError(w, http.StatusNotFound, "Channel not found")
		return
	}

	_, err = h.queries.ToggleSubscriptionFavorite(ctx, db.ToggleSubscriptionFavoriteParams{
		UserID: userID, SourceID: source.ID, IsFavorite: reqBody.IsFavorite,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to toggle favorite")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// respondError はエラーレスポンスを返す
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

