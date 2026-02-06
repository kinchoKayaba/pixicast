package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env.dev"); err != nil {
		log.Println("⚠️ .env.dev not found, using system environment variables")
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

	fmt.Println("🗑️ Starting anonymous user data cleanup...")
	cutoffDate := time.Now().AddDate(0, 0, -30)
	fmt.Printf("📆 Cutoff date: %s\n\n", cutoffDate.Format("2006-01-02 15:04:05"))

	var count int64
	err = pool.QueryRow(ctx, `SELECT COUNT(DISTINCT user_id) FROM user_subscriptions WHERE updated_at < $1`, cutoffDate).Scan(&count)
	if err != nil {
		log.Fatalf("Failed to count target users: %v", err)
	}

	fmt.Printf("🔍 Found %d users with old subscriptions\n", count)
	if count == 0 {
		fmt.Println("✅ No old subscriptions found. Nothing to clean up.")
		return
	}

	fmt.Print("\n⚠️ Do you want to delete these old subscriptions? (yes/no): ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "yes" {
		fmt.Println("❌ Cleanup cancelled.")
		return
	}

	fmt.Println("\n🗑️ Deleting old subscriptions...")

	result, err := pool.Exec(ctx, `DELETE FROM events WHERE source_id IN (SELECT DISTINCT us.source_id FROM user_subscriptions us LEFT JOIN user_subscriptions us2 ON us.source_id = us2.source_id AND us2.updated_at >= $1 WHERE us.updated_at < $1 AND us2.source_id IS NULL)`, cutoffDate)
	if err != nil {
		log.Fatalf("Failed to delete events: %v", err)
	}
	fmt.Printf("  ✓ Deleted %d events\n", result.RowsAffected())

	result, err = pool.Exec(ctx, `DELETE FROM user_subscriptions WHERE updated_at < $1`, cutoffDate)
	if err != nil {
		log.Fatalf("Failed to delete subscriptions: %v", err)
	}
	fmt.Printf("  ✓ Deleted %d subscriptions\n", result.RowsAffected())

	result, err = pool.Exec(ctx, `DELETE FROM sources WHERE id NOT IN (SELECT DISTINCT source_id FROM user_subscriptions)`)
	if err != nil {
		log.Fatalf("Failed to delete orphan sources: %v", err)
	}
	fmt.Printf("  ✓ Deleted %d orphan sources\n", result.RowsAffected())

	fmt.Println("\n✅ Cleanup completed successfully!")
}







