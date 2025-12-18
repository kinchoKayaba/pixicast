package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	// 生成されたコードのインポート
	"github.com/kinchoKayaba/pixicast/backend/db" // ★sqlcが作ったコード
	pixicastv1 "github.com/kinchoKayaba/pixicast/backend/gen/pixicast/v1"
	"github.com/kinchoKayaba/pixicast/backend/gen/pixicast/v1/pixicastv1connect"
	"github.com/kinchoKayaba/pixicast/backend/internal/auth"
	"github.com/kinchoKayaba/pixicast/backend/internal/http/handlers"
	"github.com/kinchoKayaba/pixicast/backend/internal/podcast"
	"github.com/kinchoKayaba/pixicast/backend/internal/twitch"
	"github.com/kinchoKayaba/pixicast/backend/internal/youtube"
)

// サーバー構造体
// 生のDB接続ではなく、sqlcが生成した「Queries」を持ちます
type TimelineServer struct {
	queries      *db.Queries
	youtube      *youtube.Client
	firebaseAuth *auth.FirebaseAuth
}

// parseDuration は ISO 8601 duration (PT1H30M15S) を "01:30:15" 形式に変換
func parseDuration(isoDuration string) string {
	if isoDuration == "" {
		return "00:00"
	}

	re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?`)
	matches := re.FindStringSubmatch(isoDuration)
	if matches == nil {
		return "00:00"
	}

	hours, _ := strconv.Atoi(matches[1])
	minutes, _ := strconv.Atoi(matches[2])
	seconds, _ := strconv.Atoi(matches[3])

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// タイムライン取得
func (s *TimelineServer) GetTimeline(
	ctx context.Context,
	req *connect.Request[pixicastv1.GetTimelineRequest],
) (*connect.Response[pixicastv1.GetTimelineResponse], error) {
	log.Printf("GetTimeline called for date: %s, youtube_channel_ids: %v, before_time: %s, limit: %d", 
		req.Msg.Date, req.Msg.YoutubeChannelIds, req.Msg.BeforeTime, req.Msg.Limit)

	// リクエストパラメータの処理
	limit := int32(req.Msg.Limit)
	if limit <= 0 {
		limit = 50 // デフォルト50件
	}
	if limit > 100 {
		limit = 100 // 最大100件
	}

	// before_timeの処理
	var beforeTime pgtype.Timestamptz
	if req.Msg.BeforeTime != "" {
		t, err := time.Parse(time.RFC3339, req.Msg.BeforeTime)
		if err != nil {
			log.Printf("Failed to parse before_time: %v", err)
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid before_time format"))
		}
		beforeTime = pgtype.Timestamptz{Time: t, Valid: true}
	} else {
		beforeTime = pgtype.Timestamptz{Valid: false}
	}

	// 認証: user_idを取得
	authHeader := req.Header().Get("Authorization")
	if authHeader == "" {
		log.Printf("❌ GetTimeline: Authorization header is missing")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	idToken, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		log.Printf("❌ GetTimeline: Failed to extract token: %v", err)
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid authorization header"))
	}

	token, err := s.firebaseAuth.VerifyIDToken(ctx, idToken)
	if err != nil {
		log.Printf("❌ GetTimeline: Failed to verify token: %v", err)
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token"))
	}

	userID := auth.GetUserIDFromToken(token)
	log.Printf("✅ GetTimeline: Authenticated user_id=%d", userID)

	// 1. DBからデータを取得 (SQL実行) - 新スキーマのListTimelineを使用
	// limit+1件取得して、has_moreを判定
	
	// チャンネルIDの配列を準備（空配列の場合はnilを渡す）
	var channelIds []string
	if len(req.Msg.YoutubeChannelIds) > 0 {
		channelIds = req.Msg.YoutubeChannelIds
	}
	
	timelineData, err := s.queries.ListTimeline(ctx, db.ListTimelineParams{
		UserID:     userID,
		Column2:    beforeTime,
		Limit:      limit + 1, // 1件多く取得してhas_moreを判定
		ChannelIds: channelIds,
	})
	if err != nil {
		log.Printf("Failed to fetch timeline: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error"))
	}
	log.Printf("📊 DB timeline events fetched: %d (requested: %d), channel_ids: %v", len(timelineData), limit, channelIds)

	// 2. DBの型(db.ListTimelineRow) を gRPCの型(pixicastv1.Program) に変換
	var responsePrograms []*pixicastv1.Program
	for _, event := range timelineData {
		// 放送中かどうかの簡易判定
		now := time.Now()
		isLive := event.Type == "live" && 
			event.StartAt.Valid && 
			now.After(event.StartAt.Time) &&
			(!event.EndAt.Valid || now.Before(event.EndAt.Time))

		// NULL許容フィールドの処理
		imageUrl := ""
		if event.ImageUrl.Valid {
			imageUrl = event.ImageUrl.String
		}
		description := ""
		if event.Description.Valid {
			description = event.Description.String
		}
		channelTitle := ""
		if event.SourceDisplayName.Valid {
			channelTitle = event.SourceDisplayName.String
		}
		channelThumbnailUrl := ""
		if event.SourceThumbnailUrl.Valid {
			channelThumbnailUrl = event.SourceThumbnailUrl.String
		}

		// start_at または published_at を使用
		startAt := ""
		publishedAt := ""
		if event.StartAt.Valid {
			startAt = event.StartAt.Time.Format(time.RFC3339)
		} else if event.PublishedAt.Valid {
			startAt = event.PublishedAt.Time.Format(time.RFC3339)
			publishedAt = event.PublishedAt.Time.Format(time.RFC3339)
		}

		endAt := ""
		if event.EndAt.Valid {
			endAt = event.EndAt.Time.Format(time.RFC3339)
		} else if event.PublishedAt.Valid {
			endAt = event.PublishedAt.Time.Format(time.RFC3339)
		}

		// metricsから再生回数を取得
		viewCount := int64(0)
		if len(event.Metrics) > 0 {
			var metricsData map[string]interface{}
			if err := json.Unmarshal(event.Metrics, &metricsData); err == nil {
				if views, ok := metricsData["views"].(float64); ok {
					viewCount = int64(views)
				}
			}
		}

		// durationの取得
		duration := ""
		if event.Duration.Valid {
			duration = event.Duration.String
		}

		responsePrograms = append(responsePrograms, &pixicastv1.Program{
			Id:                  event.ID.String(),
			Title:               event.Title,
			StartAt:             startAt,
			EndAt:               endAt,
			PlatformName:        event.PlatformID,
			ImageUrl:            imageUrl,
			LinkUrl:             event.Url,
			IsLive:              isLive,
			ChannelTitle:        channelTitle,
			Description:         description,
			Duration:            duration,
			PublishedAt:         publishedAt,
			ViewCount:           viewCount,
			ChannelThumbnailUrl: channelThumbnailUrl,
		})
	}

	// has_moreとnext_cursorの設定
	hasMore := false
	nextCursor := ""
	
	// limit+1件取得した場合、最後の1件を除いてhas_more=trueに設定
	if len(timelineData) > int(limit) {
		hasMore = true
		responsePrograms = responsePrograms[:limit] // 最後の1件を除く
		
		// 最後のプログラムの時刻をnext_cursorとして設定
		lastProgram := responsePrograms[len(responsePrograms)-1]
		if lastProgram.PublishedAt != "" {
			nextCursor = lastProgram.PublishedAt
		} else {
			nextCursor = lastProgram.StartAt
		}
	}

	log.Printf("📤 Returning %d programs, has_more: %v", len(responsePrograms), hasMore)

	// レスポンスを返す（DBクエリで既にソート済み）
	return connect.NewResponse(&pixicastv1.GetTimelineResponse{
		Programs:   responsePrograms,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}), nil
}

func (s *TimelineServer) SearchYouTubeLive(
	ctx context.Context,
	req *connect.Request[pixicastv1.SearchYouTubeLiveRequest],
) (*connect.Response[pixicastv1.SearchYouTubeLiveResponse], error) {
	log.Printf("SearchYouTubeLive called with query: %s, max_results: %d", req.Msg.Query, req.Msg.MaxResults)

	// デフォルト値の設定
	maxResults := int64(req.Msg.MaxResults)
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 50 {
		maxResults = 50
	}

	// YouTube APIでライブ配信を検索
	streams, err := s.youtube.SearchLiveStreams(ctx, req.Msg.Query, maxResults)
	if err != nil {
		log.Printf("Failed to search YouTube live streams: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("YouTube API error"))
	}

	// レスポンスに変換
	var responseStreams []*pixicastv1.YouTubeLiveStream
	for _, stream := range streams {
		thumbnailUrl := ""
		if stream.Snippet.Thumbnails != nil && stream.Snippet.Thumbnails.High != nil {
			thumbnailUrl = stream.Snippet.Thumbnails.High.Url
		}

		responseStreams = append(responseStreams, &pixicastv1.YouTubeLiveStream{
			VideoId:      stream.Id.VideoId,
			Title:        stream.Snippet.Title,
			ChannelTitle: stream.Snippet.ChannelTitle,
			Description:  stream.Snippet.Description,
			ThumbnailUrl: thumbnailUrl,
			PublishedAt:  stream.Snippet.PublishedAt,
		})
	}

	return connect.NewResponse(&pixicastv1.SearchYouTubeLiveResponse{
		Streams: responseStreams,
	}), nil
}

func main() {
	// 環境変数ファイルを読み込む（ローカル開発用）
	// Cloud Runなどの本番環境では環境変数を直接設定するので、.envファイルは不要
	// GO_ENV が production なら .env.production、それ以外は .env.dev を試みる
	envFile := ".env.dev"
	if os.Getenv("GO_ENV") == "production" {
		envFile = ".env.production"
	}
	// ファイルが存在しなくてもエラーにしない（本番環境では環境変数が直接設定される）
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("Info: .env file not loaded (%s), using system environment variables", envFile)
	} else {
		log.Printf("✅ Loaded environment from %s", envFile)
	}

	// YouTube API クライアントの初期化
	youtubeAPIKey := os.Getenv("YOUTUBE_API_KEY")
	if youtubeAPIKey == "" {
		log.Fatal("YOUTUBE_API_KEY environment variable is not set")
	}

	youtubeClient, err := youtube.NewClient(youtubeAPIKey)
	if err != nil {
		log.Fatalf("Failed to create YouTube client: %v", err)
	}
	fmt.Println("✅ YouTube API client initialized successfully!")

	// Twitch API クライアントの初期化
	twitchClient := twitch.NewClient()
	fmt.Println("✅ Twitch API client initialized successfully!")

	// Podcast クライアントの初期化
	podcastClient := podcast.NewClient()
	fmt.Println("✅ Podcast client initialized successfully!")

	// Firebase Auth の初期化
	firebaseAuth, err := auth.NewFirebaseAuth(context.Background())
	if err != nil {
		log.Fatalf("Failed to initialize Firebase Auth: %v", err)
	}
	fmt.Println("✅ Firebase Auth initialized successfully!")

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// DB接続
	pool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// 疎通確認
	if err = pool.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("✅ Connected to CockroachDB successfully!")

	// ★ここがポイント: DB接続を使って sqlc の Queries を作成
	queries := db.New(pool)

	// サーバーに渡す
	server := &TimelineServer{
		queries:      queries,
		youtube:      youtubeClient,
		firebaseAuth: firebaseAuth,
	}

	path, handler := pixicastv1connect.NewTimelineServiceHandler(server)
	
	// CORSミドルウェアを追加
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, Authorization")
			w.Header().Set("Access-Control-Expose-Headers", "Connect-Protocol-Version, Connect-Timeout-Ms")
			
			// プリフライトリクエストに対応
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			h.ServeHTTP(w, r)
		})
	}
	
	// Subscription ハンドラを作成
	subscriptionHandler := handlers.NewSubscriptionHandler(queries, youtubeClient, twitchClient, podcastClient, firebaseAuth)
	
	mux := http.NewServeMux()
	mux.Handle(path, corsHandler(handler))
	
	// REST APIエンドポイント
	mux.HandleFunc("/v1/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		// CORS処理
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		if r.Method == "GET" {
			subscriptionHandler.ListSubscriptions(w, r)
			return
		}
		
		if r.Method == "POST" {
			subscriptionHandler.CreateSubscription(w, r)
			return
		}
		
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	
	// GET /v1/me - ユーザー情報とプラン情報を取得
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		// CORS処理
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		if r.Method == "GET" {
			subscriptionHandler.GetMe(w, r)
			return
		}
		
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	
	// DELETE /v1/subscriptions/{channelId}
	// POST /v1/subscriptions/{channelId}/favorite
	mux.HandleFunc("/v1/subscriptions/", func(w http.ResponseWriter, r *http.Request) {
		// CORS処理
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// POST で /favorite で終わる場合はToggleFavorite
		if r.Method == "POST" && len(r.URL.Path) > 0 && r.URL.Path[len(r.URL.Path)-9:] == "/favorite" {
			subscriptionHandler.ToggleFavorite(w, r)
			return
		}
		if r.Method == "DELETE" {
			subscriptionHandler.DeleteSubscription(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	fmt.Printf("Starting Pixicast Server (Timeline Mode) on %s ...\n", addr)
	err = http.ListenAndServe(
		addr,
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}