package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kinchoKayaba/pixicast/backend/internal/twitch"
)

func updateLiveStatus() {
	log.Println("🔄 Starting live status update...")

	// 環境変数読み込み
	if err := godotenv.Load(".env.dev"); err != nil {
		log.Printf("Warning: .env.dev not loaded (%v)", err)
	}

	// DB接続
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("❌ DATABASE_URL not set")
	}

	pool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Twitch クライアント初期化
	twitchClientID := os.Getenv("TWITCH_CLIENT_ID")
	twitchClientSecret := os.Getenv("TWITCH_CLIENT_SECRET")
	if twitchClientID == "" || twitchClientSecret == "" {
		log.Fatal("❌ TWITCH_CLIENT_ID or TWITCH_CLIENT_SECRET not set")
	}

	twitchClient := twitch.NewClient()

	// 現在「live」タイプのTwitchイベントを取得
	rows, err := pool.Query(ctx, `
		SELECT e.id, e.external_event_id, e.source_id, e.title, e.start_at,
		       s.external_id as twitch_user_id
		FROM events e
		JOIN sources s ON e.source_id = s.id
		WHERE e.platform_id = 'twitch'
		  AND e.type = 'live'
		  AND (e.end_at IS NULL OR e.end_at > NOW())
	`)
	if err != nil {
		log.Fatalf("❌ Failed to query live events: %v", err)
	}
	defer rows.Close()

	type LiveEvent struct {
		ID              pgtype.UUID
		ExternalEventID string
		SourceID        pgtype.UUID
		Title           string
		StartAt         pgtype.Timestamptz
		TwitchUserID    string
	}

	var liveEvents []LiveEvent
	for rows.Next() {
		var event LiveEvent
		if err := rows.Scan(&event.ID, &event.ExternalEventID, &event.SourceID, &event.Title, &event.StartAt, &event.TwitchUserID); err != nil {
			log.Printf("⚠️ Failed to scan row: %v", err)
			continue
		}
		liveEvents = append(liveEvents, event)
	}

	if len(liveEvents) == 0 {
		log.Println("✅ No active live streams to check")
		return
	}

	log.Printf("📺 Checking %d live events...", len(liveEvents))

	// Twitchユーザーごとにグループ化
	userEvents := make(map[string][]LiveEvent)
	for _, event := range liveEvents {
		userEvents[event.TwitchUserID] = append(userEvents[event.TwitchUserID], event)
	}

	updatedCount := 0

	// 各ユーザーの配信状況をチェック
	for twitchUserID, events := range userEvents {
		log.Printf("🔍 Checking Twitch user: %s", twitchUserID)

		// 現在配信中のストリームを取得
		streams, err := twitchClient.GetStreams(ctx, twitchUserID)
		if err != nil {
			log.Printf("⚠️ Failed to get streams for user %s: %v", twitchUserID, err)
			continue
		}

		// 現在配信中のstream IDのマップを作成
		currentStreamIDs := make(map[string]bool)
		for _, stream := range streams {
			currentStreamIDs[stream.ID] = true
		}

		// DBの各イベントをチェック
		for _, event := range events {
			// このイベントが現在配信中かチェック
			if currentStreamIDs[event.ExternalEventID] {
				log.Printf("✅ Still live: %s", event.Title)
				continue
			}

			// 配信が終了している場合、DBを更新
			log.Printf("🔴 Stream ended: %s", event.Title)
			
			// end_atを現在時刻に設定し、typeを"video"に変更
			_, err := pool.Exec(ctx, `
				UPDATE events
				SET type = 'video',
				    end_at = NOW(),
				    updated_at = NOW()
				WHERE id = $1
			`, event.ID)
			if err != nil {
				log.Printf("⚠️ Failed to update event %s: %v", event.Title, err)
				continue
			}

			updatedCount++
			log.Printf("✅ Updated: %s (live -> video)", event.Title)
		}
	}

	log.Printf("✅ Live status update completed. Updated %d events.", updatedCount)
}

func main() {
	updateLiveStatus()
}

