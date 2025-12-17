package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kinchoKayaba/pixicast/backend/db"
	"github.com/kinchoKayaba/pixicast/backend/internal/ingest"
	"github.com/kinchoKayaba/pixicast/backend/internal/youtube"
)

func main() {
	log.Println("🚀 Starting video fetch batch job...")

	// .env.dev ファイルを読み込み
	if err := godotenv.Load(".env.dev"); err != nil {
		log.Printf("Warning: .env.dev file not found: %v", err)
	}

	// データベース接続
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	// YouTube クライアント
	youtubeAPIKey := os.Getenv("YOUTUBE_API_KEY")
	if youtubeAPIKey == "" {
		log.Fatal("YOUTUBE_API_KEY environment variable is not set")
	}

	youtubeClient, err := youtube.NewClient(youtubeAPIKey)
	if err != nil {
		log.Fatalf("Failed to create YouTube client: %v", err)
	}

	// すべてのソース（チャンネル）を取得
	sources, err := queries.ListSources(ctx, 1000) // 最大1000チャンネル
	if err != nil {
		log.Fatalf("Failed to list sources: %v", err)
	}

	log.Printf("📺 Found %d sources to fetch", len(sources))

	// 各ソースから動画を取得（2025/1/1以降の全動画）
	publishedAfter := "2025-01-01T00:00:00Z" // RFC3339形式
	totalVideos := 0
	
	for _, source := range sources {
		if source.PlatformID != "youtube" {
			continue
		}

		displayName := "Unknown"
		if source.DisplayName.Valid {
			displayName = source.DisplayName.String
		}
		log.Printf("Fetching videos for channel: %s (%s) since 2025/1/1", displayName, source.ExternalID)

		// 2025/1/1以降の全動画を取得
		err := ingest.FetchAndSaveChannelVideosSince(
			ctx,
			queries,
			youtubeClient,
			source.ID,
			source.ExternalID,
			0, // 制限なし（全動画取得）
			publishedAfter,
		)
		if err != nil {
			log.Printf("❌ Failed to fetch videos for channel %s: %v", source.ExternalID, err)
			continue
		}

		totalVideos++ // カウント（後で正確な数に修正可能）

		// TODO: 最後のフェッチ時刻を更新（クエリが必要）
	}

	log.Printf("🎉 Batch job completed! Total videos saved: %d", totalVideos)
}

