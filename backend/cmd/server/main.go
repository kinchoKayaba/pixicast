package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	// 生成されたコードのインポート
	"github.com/kinchoKayaba/pixicast/backend/db" // ★sqlcが作ったコード
	pixicastv1 "github.com/kinchoKayaba/pixicast/backend/gen/pixicast/v1"
	"github.com/kinchoKayaba/pixicast/backend/gen/pixicast/v1/pixicastv1connect"
	"github.com/kinchoKayaba/pixicast/backend/internal/youtube"
)

// サーバー構造体
// 生のDB接続ではなく、sqlcが生成した「Queries」を持ちます
type TimelineServer struct {
	queries *db.Queries
	youtube *youtube.Client
}

// タイムライン取得
func (s *TimelineServer) GetTimeline(
	ctx context.Context,
	req *connect.Request[pixicastv1.GetTimelineRequest],
) (*connect.Response[pixicastv1.GetTimelineResponse], error) {
	log.Printf("GetTimeline called for date: %s, youtube_channel_ids: %v", req.Msg.Date, req.Msg.YoutubeChannelIds)

	// 1. DBからデータを取得 (SQL実行)
	programsData, err := s.queries.ListPrograms(ctx)
	if err != nil {
		log.Printf("Failed to fetch programs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error"))
	}
	log.Printf("📊 DB programs fetched: %d", len(programsData))

	// 2. DBの型(db.Program) を gRPCの型(pixicastv1.Program) に変換
	var responsePrograms []*pixicastv1.Program
	for _, p := range programsData {
		// 放送中かどうかの簡易判定 (現在時刻が start と end の間なら true)
		now := time.Now()
		isLive := now.After(p.StartAt.Time) && now.Before(p.EndAt.Time)

		// ImageUrlなどはNULL許容(pgtype.Text)なので、取り出し方に注意
		imageUrl := ""
		if p.ImageUrl.Valid {
			imageUrl = p.ImageUrl.String
		}
		linkUrl := ""
		if p.LinkUrl.Valid {
			linkUrl = p.LinkUrl.String
		}

		responsePrograms = append(responsePrograms, &pixicastv1.Program{
			Id:           p.ID.String(), // UUIDを文字列に
			Title:        p.Title,
			StartAt:      p.StartAt.Time.Format(time.RFC3339), // 時間を文字列に
			EndAt:        p.EndAt.Time.Format(time.RFC3339),
			PlatformName: p.PlatformName,
			ImageUrl:     imageUrl,
			LinkUrl:      linkUrl,
			IsLive:       isLive,
		})
	}

	// 3. YouTubeチャンネルからデータを取得
	for _, channelID := range req.Msg.YoutubeChannelIds {
		videos, err := s.youtube.GetChannelVideos(ctx, channelID, 20)
		if err != nil {
			log.Printf("Failed to get YouTube videos for channel %s: %v", channelID, err)
			continue // エラーでも他のチャンネルは続行
		}
		log.Printf("📺 YouTube videos fetched from channel %s: %d", channelID, len(videos))

		for _, video := range videos {
			thumbnailUrl := ""
			if video.Snippet.Thumbnails != nil && video.Snippet.Thumbnails.High != nil {
				thumbnailUrl = video.Snippet.Thumbnails.High.Url
			}

			// published_atをパース
			publishedAt, err := time.Parse(time.RFC3339, video.Snippet.PublishedAt)
			if err != nil {
				log.Printf("Failed to parse published_at: %v", err)
				publishedAt = time.Now()
			}

			responsePrograms = append(responsePrograms, &pixicastv1.Program{
				Id:           video.Id.VideoId,
				Title:        video.Snippet.Title,
				StartAt:      publishedAt.Format(time.RFC3339),
				EndAt:        publishedAt.Format(time.RFC3339), // YouTubeは同じ値
				PlatformName: "YouTube",
				ImageUrl:     thumbnailUrl,
				LinkUrl:      fmt.Sprintf("https://www.youtube.com/watch?v=%s", video.Id.VideoId),
				IsLive:       video.Snippet.LiveBroadcastContent == "live",
				ChannelTitle: video.Snippet.ChannelTitle,
				Description:  video.Snippet.Description,
			})
		}
	}

	// 4. 全てのプログラムを時系列順（新しい順）にソート
	sort.Slice(responsePrograms, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339, responsePrograms[i].StartAt)
		timeJ, errJ := time.Parse(time.RFC3339, responsePrograms[j].StartAt)
		if errI != nil || errJ != nil {
			return false
		}
		// 降順（新しい順）
		return timeI.After(timeJ)
	})

	return connect.NewResponse(&pixicastv1.GetTimelineResponse{
		Programs: responsePrograms,
	}), nil
}

// YouTubeライブ配信検索
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
		queries: queries,
		youtube: youtubeClient,
	}

	path, handler := pixicastv1connect.NewTimelineServiceHandler(server)
	
	// CORSミドルウェアを追加
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms")
			w.Header().Set("Access-Control-Expose-Headers", "Connect-Protocol-Version, Connect-Timeout-Ms")
			
			// プリフライトリクエストに対応
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			h.ServeHTTP(w, r)
		})
	}
	
	mux := http.NewServeMux()
	mux.Handle(path, corsHandler(handler))

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