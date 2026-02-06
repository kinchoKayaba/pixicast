package main

import (
	"context"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kinchoKayaba/pixicast/backend/db"
	"github.com/kinchoKayaba/pixicast/backend/internal/ingest"
	"github.com/kinchoKayaba/pixicast/backend/internal/podcast"
	"github.com/kinchoKayaba/pixicast/backend/internal/twitch"
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

	// Twitch クライアント
	twitchClient := twitch.NewClient()
	if twitchClient == nil {
		log.Fatal("Failed to create Twitch client")
	}

	// Podcast クライアント
	podcastClient := podcast.NewClient()

	// すべてのソース（チャンネル）を取得
	sources, err := queries.ListSources(ctx, 1000) // 最大1000チャンネル
	if err != nil {
		log.Fatalf("Failed to list sources: %v", err)
	}

	log.Printf("📺 Found %d sources to fetch", len(sources))

	// 並列処理用のカウンター
	var totalSuccess, totalFailed atomic.Int32
	
	// ワーカープール（最大10並行）
	maxWorkers := 10
	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	
	for _, source := range sources {
		wg.Add(1)
		
		// goroutineで並列処理
		go func(src db.Source) {
			defer wg.Done()
			
			// セマフォで並行数を制限
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

		displayName := "Unknown"
			if src.DisplayName.Valid {
				displayName = src.DisplayName.String
		}

			var err error
			var publishedAfter string

			switch src.PlatformID {
			case "youtube":
				// YouTube: 増分更新（前回取得時刻以降のみ）
				// 初回は過去3ヶ月分
				if src.LastFetchedAt.Valid {
					publishedAfter = src.LastFetchedAt.Time.Add(-5 * time.Minute).Format(time.RFC3339)
				} else {
					publishedAfter = time.Now().AddDate(0, -3, 0).Format(time.RFC3339) // 3ヶ月前
				}
				log.Printf("📺 [YouTube] %s (since %s)", displayName, publishedAfter)
				err = ingest.FetchAndSaveChannelVideosSince(
			ctx,
			queries,
			youtubeClient,
					src.ID,
					src.ExternalID,
					0,
					publishedAfter,
				)

			case "twitch":
				// Twitch: ライブは常時チェック、VODは直近1週間のみ
				// 前回取得時刻と1週間前の新しい方を使う
				oneWeekAgo := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
				if src.LastFetchedAt.Valid {
					lastFetched := src.LastFetchedAt.Time.Add(-5 * time.Minute).Format(time.RFC3339)
					if lastFetched > oneWeekAgo {
						publishedAfter = lastFetched
					} else {
						publishedAfter = oneWeekAgo
					}
				} else {
					publishedAfter = oneWeekAgo
				}
				log.Printf("🎮 [Twitch] %s (🔴LIVE + VOD since %s)", displayName, publishedAfter)
				err = ingest.FetchAndSaveTwitchVideosSince(
					ctx,
					queries,
					twitchClient,
					src.ID,
					src.ExternalID,
					publishedAfter,
				)

			case "podcast":
				// Podcast: 直近1週間は常にチェック（放送日から遅れて配信される場合がある）
				oneWeekAgo := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
				if src.LastFetchedAt.Valid {
					lastFetched := src.LastFetchedAt.Time.Add(-5 * time.Minute).Format(time.RFC3339)
					if lastFetched > oneWeekAgo {
						publishedAfter = lastFetched
					} else {
						publishedAfter = oneWeekAgo
					}
				} else {
					publishedAfter = time.Now().AddDate(0, -3, 0).Format(time.RFC3339) // 初回は3ヶ月前
				}
				log.Printf("🎙️ [Podcast] %s (since %s)", displayName, publishedAfter)
				err = ingest.FetchAndSavePodcastEpisodesSince(
					ctx,
					queries,
					podcastClient,
					src.ID,
					src.ExternalID,
			publishedAfter,
		)

			default:
				log.Printf("⚠️ Unknown platform: %s", src.PlatformID)
				return
			}

		if err != nil {
				log.Printf("❌ Failed to fetch content for %s (%s): %v", displayName, src.ExternalID, err)
				totalFailed.Add(1)
				return
			}

			// 取得成功: last_fetched_atを更新
			_, updateErr := queries.UpdateSourceFetchStatus(ctx, db.UpdateSourceFetchStatusParams{
				ID:          src.ID,
				FetchStatus: "ok",
			})
			if updateErr != nil {
				log.Printf("⚠️ Failed to update last_fetched_at for %s: %v", displayName, updateErr)
			}

			totalSuccess.Add(1)
		}(source)
	}
	
	// すべてのgoroutineの完了を待つ
	wg.Wait()

	log.Printf("🎉 Batch job completed! Success: %d, Failed: %d", totalSuccess.Load(), totalFailed.Load())
}

