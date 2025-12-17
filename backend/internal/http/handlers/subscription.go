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
	"github.com/kinchoKayaba/pixicast/backend/internal/ingest"
	"github.com/kinchoKayaba/pixicast/backend/internal/youtube"
)

// SubscriptionHandler は購読関連のハンドラ
type SubscriptionHandler struct {
	queries *db.Queries
	youtube *youtube.Client
}

// NewSubscriptionHandler はハンドラを作成
func NewSubscriptionHandler(queries *db.Queries, youtubeClient *youtube.Client) *SubscriptionHandler {
	return &SubscriptionHandler{
		queries: queries,
		youtube: youtubeClient,
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
}

// ErrorResponse はエラーレスポンス
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateSubscription は購読登録API
// POST /v1/subscriptions
func (h *SubscriptionHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// リクエスト解析
	var req CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// バリデーション
	if req.Platform != "youtube" {
		respondError(w, http.StatusBadRequest, "only youtube platform is supported")
		return
	}
	if req.Input == "" {
		respondError(w, http.StatusBadRequest, "input is required")
		return
	}

	// 入力正規化
	channelID, handle, err := h.normalizeInput(req.Input)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("Normalized input: channelID=%s, handle=%s", channelID, handle)

	// handleの場合はchannelIDに解決
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

	// チャンネル詳細を取得
	details, err := h.youtube.GetChannelDetails(ctx, channelID)
	if err != nil {
		log.Printf("Failed to get channel details for %s: %v", channelID, err)
		respondError(w, http.StatusNotFound, "channel not found")
		return
	}

	// handleが空の場合はAPIから取得した値を使用
	if handle == "" && details.Handle != "" {
		handle = details.Handle
	}

	log.Printf("Channel details: %+v", details)

	// DB操作
	subscription, err := h.upsertSubscription(ctx, 1, req.Platform, details) // user_id=1 固定
	if err != nil {
		log.Printf("Failed to upsert subscription: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}

	// 非同期取り込みキック（スタブ）
	go h.enqueueIngest(subscription.SourceID)

	// レスポンス
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateSubscriptionResponse{
		Subscription: SubscriptionData{
			UserID:       subscription.UserID,
			Platform:     req.Platform,
			SourceID:     subscription.SourceID,
			ChannelID:    channelID,
			Handle:       handle,
			DisplayName:  details.DisplayName,
			ThumbnailURL: details.ThumbnailURL,
			Enabled:      subscription.Enabled,
		},
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
func (h *SubscriptionHandler) upsertSubscription(
	ctx context.Context,
	userID int64,
	platform string,
	details *youtube.ChannelDetails,
) (*subscriptionResult, error) {
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

	// チャンネル追加後、すぐに動画を取得してDBに保存
	go func() {
		ingestCtx := context.Background()
		if err := ingest.FetchAndSaveChannelVideos(
			ingestCtx,
			h.queries,
			h.youtube,
			source.ID,
			details.ChannelID,
			20, // 最大20動画
		); err != nil {
			log.Printf("Failed to fetch videos for channel %s: %v", details.ChannelID, err)
		}
	}()

	return &subscriptionResult{
		UserID:   userID,
		SourceID: source.ID.String(),
		Enabled:  subscription.Enabled,
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

	// TODO: 認証実装後はリクエストからuser_idを取得
	userID := int64(1) // 暫定: user_id=1 固定

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
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodDelete {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
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

	// 固定ユーザーID（後でJWTから取得）
	const userID int64 = 1
	ctx := context.Background()

	// チャンネルを検索
	source, err := h.queries.GetSourceByExternalID(ctx, db.GetSourceByExternalIDParams{
		PlatformID: "youtube",
		ExternalID: channelID,
	})
	if err != nil {
		log.Printf("Failed to find source: %v", err)
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

// respondError はエラーレスポンスを返す
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

