package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kinchoKayaba/pixicast/backend/db"
)

func main() {
	log.Println("🔄 Starting priority calculation batch...")

	// Database接続
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	// 優先度を計算
	start := time.Now()
	if err := queries.CalculateSourcePriority(ctx); err != nil {
		log.Fatalf("Failed to calculate source priority: %v", err)
	}

	// 統計情報を取得
	stats, err := queries.GetSourcePriorityStats(ctx)
	if err != nil {
		log.Printf("⚠️  Failed to get priority stats: %v", err)
	} else {
		log.Println("📊 Priority Statistics:")
		for _, stat := range stats {
			log.Printf("  - %s: %d sources (avg %.2f%% popularity, update every %d min)",
				stat.PriorityLevel,
				stat.SourceCount,
				float64(stat.AvgPopularity)*100,
				stat.UpdateIntervalMinutes,
			)
		}
	}

	elapsed := time.Since(start)
	log.Printf("✅ Priority calculation completed in %v", elapsed)
}
