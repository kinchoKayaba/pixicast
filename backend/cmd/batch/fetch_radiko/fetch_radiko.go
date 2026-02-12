package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kinchoKayaba/pixicast/backend/db"
	"github.com/kinchoKayaba/pixicast/backend/internal/ingest"
	"github.com/kinchoKayaba/pixicast/backend/internal/radiko"
)

func main() {
	log.Println("📻 Starting Radiko fetch batch job...")

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

	// Radikoクライアント作成（デフォルト: 東京）
	areaID := os.Getenv("RADIKO_AREA_ID")
	if areaID == "" {
		areaID = "JP13" // 東京
	}

	radikoClient := radiko.NewClient(areaID)

	// Radiko対応のソースを取得
	sources, err := queries.ListSourcesByPlatform(ctx, "radiko")
	if err != nil {
		log.Printf("⚠️  Failed to list Radiko sources: %v (まだRadiko局が登録されていない可能性があります)", err)
		log.Println("📻 エリアの全ラジオ局を取得してデモ実行します...")

		// デモ: エリアの全ラジオ局を取得
		stations, err := radikoClient.GetStations(ctx, areaID)
		if err != nil {
			log.Fatalf("Failed to get stations: %v", err)
		}

		log.Printf("✅ Found %d stations in area %s", len(stations), areaID)
		for _, station := range stations {
			log.Printf("  - %s (%s)", station.Name, station.ID)
		}

		// 最初の局の番組を取得してみる
		if len(stations) > 0 {
			testStation := stations[0]
			log.Printf("\n📻 Testing with station: %s (%s)", testStation.Name, testStation.ID)

			// 今日の番組を取得
			programs, err := radikoClient.GetProgramsByDate(ctx, testStation.ID, time.Now())
			if err != nil {
				log.Fatalf("Failed to get programs: %v", err)
			}

			log.Printf("✅ Found %d programs for today", len(programs))
			// 最初の5番組を表示
			for i, prog := range programs {
				if i >= 5 {
					break
				}
				log.Printf("  %s - %s: %s",
					prog.StartTime.Format("15:04"),
					prog.EndTime.Format("15:04"),
					prog.Title,
				)
			}
		}

		return
	}

	log.Printf("📻 Found %d Radiko sources to fetch", len(sources))

	// 各ソース（ラジオ局）の番組を取得
	for _, source := range sources {
		displayName := "Unknown"
		if source.DisplayName.Valid {
			displayName = source.DisplayName.String
		}

		log.Printf("📻 Fetching programs for: %s (%s)", displayName, source.ExternalID)

		// 前回取得時刻以降、または初回は1週間前から
		var since string
		if source.LastFetchedAt.Valid {
			since = source.LastFetchedAt.Time.Add(-5 * time.Minute).Format(time.RFC3339)
		} else {
			since = time.Now().AddDate(0, 0, -7).Format(time.RFC3339) // 1週間前
		}

		err := ingest.FetchAndSaveRadikoPrograms(
			ctx,
			queries,
			radikoClient,
			source.ID,
			source.ExternalID,
			since,
		)

		if err != nil {
			log.Printf("❌ Failed to fetch programs for %s: %v", displayName, err)
			continue
		}

		// 取得成功: last_fetched_atを更新
		_, updateErr := queries.UpdateSourceFetchStatus(ctx, db.UpdateSourceFetchStatusParams{
			ID:          source.ID,
			FetchStatus: "ok",
		})
		if updateErr != nil {
			log.Printf("⚠️  Failed to update last_fetched_at for %s: %v", displayName, updateErr)
		}
	}

	log.Println("🎉 Radiko fetch batch job completed!")
}
