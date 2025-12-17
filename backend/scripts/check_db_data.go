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
	// 環境変数読み込み
	if err := godotenv.Load(".env.dev"); err != nil {
		log.Printf("Info: .env.dev not loaded, using system environment variables")
	}

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

	ctx := context.Background()

	// チャンネル数を確認
	rows, err := pool.Query(ctx, "SELECT COUNT(*) FROM sources")
	if err != nil {
		log.Fatalf("Failed to query sources: %v", err)
	}
	defer rows.Close()
	
	var sourceCount int
	if rows.Next() {
		rows.Scan(&sourceCount)
	}
	fmt.Printf("📊 Total channels (sources): %d\n\n", sourceCount)

	// ユーザーごとの購読数
	rows, err = pool.Query(ctx, "SELECT user_id, COUNT(*) FROM user_subscriptions GROUP BY user_id")
	if err != nil {
		log.Fatalf("Failed to query subscriptions: %v", err)
	}
	defer rows.Close()
	
	fmt.Println("👥 Subscriptions per user:")
	for rows.Next() {
		var userID int64
		var count int
		rows.Scan(&userID, &count)
		fmt.Printf("  User ID %d: %d channels\n", userID, count)
	}
	fmt.Println()

	// イベント数（動画数）
	rows, err = pool.Query(ctx, "SELECT COUNT(*) FROM events")
	if err != nil {
		log.Fatalf("Failed to query events: %v", err)
	}
	defer rows.Close()
	
	var eventCount int
	if rows.Next() {
		rows.Scan(&eventCount)
	}
	fmt.Printf("🎬 Total videos (events): %d\n\n", eventCount)

	// チャンネルごとの動画数
	rows, err = pool.Query(ctx, `
		SELECT s.display_name, s.external_id, COUNT(e.id) 
		FROM sources s 
		LEFT JOIN events e ON s.id = e.source_id 
		GROUP BY s.id, s.display_name, s.external_id
		ORDER BY COUNT(e.id) DESC
	`)
	if err != nil {
		log.Fatalf("Failed to query events per channel: %v", err)
	}
	defer rows.Close()
	
	fmt.Println("📺 Videos per channel:")
	for rows.Next() {
		var displayName, externalID string
		var count int
		rows.Scan(&displayName, &externalID, &count)
		fmt.Printf("  %s (%s): %d videos\n", displayName, externalID, count)
	}
	fmt.Println()

	// 最近の動画5件
	rows, err = pool.Query(ctx, `
		SELECT title, published_at 
		FROM events 
		ORDER BY published_at DESC 
		LIMIT 5
	`)
	if err != nil {
		log.Fatalf("Failed to query recent events: %v", err)
	}
	defer rows.Close()
	
	fmt.Println("🎥 Recent 5 videos:")
	for rows.Next() {
		var title string
		var publishedAt interface{}
		rows.Scan(&title, &publishedAt)
		fmt.Printf("  %v - %s\n", publishedAt, title)
	}
}

