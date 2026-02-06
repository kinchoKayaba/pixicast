package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// .env.dev から環境変数を読み込む
	if err := godotenv.Load(".env.dev"); err != nil {
		log.Printf("Warning: .env.dev not found, using system environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// イベント数を確認
	var totalCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&totalCount)
	if err != nil {
		log.Fatalf("Failed to count events: %v", err)
	}
	fmt.Printf("📊 Total events in DB: %d\n\n", totalCount)

	// 日付別の件数を確認
	rows, err := pool.Query(ctx, `
		SELECT 
			DATE(COALESCE(published_at, start_at)) as date,
			COUNT(*) as count
		FROM events
		WHERE COALESCE(published_at, start_at) >= '2025-01-01'
		GROUP BY DATE(COALESCE(published_at, start_at))
		ORDER BY date DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatalf("Failed to query events: %v", err)
	}
	defer rows.Close()

	fmt.Println("📅 Recent events (2025/1/1以降):")
	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		fmt.Printf("  %s: %d videos\n", date, count)
	}

	// 最新のイベントを確認
	var latestTitle string
	var latestDate string
	err = pool.QueryRow(ctx, `
		SELECT title, COALESCE(published_at, start_at)::text
		FROM events
		ORDER BY COALESCE(published_at, start_at) DESC
		LIMIT 1
	`).Scan(&latestTitle, &latestDate)
	if err != nil {
		log.Printf("Failed to get latest event: %v", err)
	} else {
		fmt.Printf("\n🎬 Latest event: %s (%s)\n", latestTitle, latestDate)
	}

	// 最古のイベントを確認
	var oldestTitle string
	var oldestDate string
	err = pool.QueryRow(ctx, `
		SELECT title, COALESCE(published_at, start_at)::text
		FROM events
		ORDER BY COALESCE(published_at, start_at) ASC
		LIMIT 1
	`).Scan(&oldestTitle, &oldestDate)
	if err != nil {
		log.Printf("Failed to get oldest event: %v", err)
	} else {
		fmt.Printf("🎬 Oldest event: %s (%s)\n", oldestTitle, oldestDate)
	}
}

