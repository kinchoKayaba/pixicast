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
	godotenv.Load(".env.dev")
	
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	
	rows, err := pool.Query(context.Background(), `
		SELECT platform_id, title, url 
		FROM events 
		WHERE published_at > NOW() - INTERVAL '1 day'
		ORDER BY published_at DESC 
		LIMIT 20
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	
	fmt.Println("📊 Recent events (last 24 hours):\n")
	for rows.Next() {
		var platform, title, url string
		rows.Scan(&platform, &title, &url)
		
		urlStatus := "✅"
		if url == "" {
			urlStatus = "❌ EMPTY"
		} else if url[0] != 'h' {
			urlStatus = "⚠️  RELATIVE"
		}
		
		fmt.Printf("[%s] %s %s\n", platform, urlStatus, title)
		fmt.Printf("     URL: %s\n\n", url)
	}
}

