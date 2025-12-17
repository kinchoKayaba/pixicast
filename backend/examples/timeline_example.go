package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kinchoKayaba/pixicast/backend/db"
)

// TimelineExample は新しいスキーマを使ったタイムライン取得の例
func TimelineExample(dbURL string) error {
	ctx := context.Background()

	// DB接続
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	// ============================================================================
	// 1. チャンネル登録（購読）
	// ============================================================================
	fmt.Println("=== 1. チャンネル登録 ===")

	// ソースをupsert
	source, err := queries.UpsertSource(ctx, db.UpsertSourceParams{
		PlatformID:        "youtube",
		ExternalID:        "UCxxxxxxxxxxxx",
		Handle:            pgtype.Text{String: "junchannel", Valid: true},
		DisplayName:       pgtype.Text{String: "Jun Channel", Valid: true},
		ThumbnailUrl:      pgtype.Text{String: "https://example.com/thumb.jpg", Valid: true},
		UploadsPlaylistID: pgtype.Text{String: "UUxxxxxxxxxxxx", Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert source: %w", err)
	}
	fmt.Printf("✅ Source created: %s (%s)\n", source.DisplayName.String, source.ID)

	// ユーザー購読をupsert
	subscription, err := queries.UpsertUserSubscription(ctx, db.UpsertUserSubscriptionParams{
		UserID:   1,
		SourceID: source.ID,
		Enabled:  true,
		Priority: 0,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert subscription: %w", err)
	}
	fmt.Printf("✅ Subscription created: user_id=%d, source_id=%s\n", subscription.UserID, subscription.SourceID)

	// ============================================================================
	// 2. イベント登録（動画/配信）
	// ============================================================================
	fmt.Println("\n=== 2. イベント登録 ===")

	// ライブ配信イベント
	liveEvent, err := queries.UpsertEvent(ctx, db.UpsertEventParams{
		PlatformID:      "youtube",
		SourceID:        source.ID,
		ExternalEventID: "live123",
		Type:            "live",
		Title:           "【LIVE】今日のゲーム配信",
		Description:     pgtype.Text{String: "今日も楽しく配信します！", Valid: true},
		StartAt:         pgtype.Timestamptz{Time: time.Now().Add(-10 * time.Minute), Valid: true},
		EndAt:           pgtype.Timestamptz{Valid: false}, // 配信中は終了時刻なし
		PublishedAt:     pgtype.Timestamptz{Time: time.Now().Add(-10 * time.Minute), Valid: true},
		Url:             "https://www.youtube.com/watch?v=live123",
		ImageUrl:        pgtype.Text{String: "https://example.com/live.jpg", Valid: true},
		Metrics:         []byte(`{"viewers": 1234, "likes": 567}`),
	})
	if err != nil {
		return fmt.Errorf("failed to upsert live event: %w", err)
	}
	fmt.Printf("✅ Live event created: %s\n", liveEvent.Title)

	// 予定配信イベント
	scheduledEvent, err := queries.UpsertEvent(ctx, db.UpsertEventParams{
		PlatformID:      "youtube",
		SourceID:        source.ID,
		ExternalEventID: "scheduled456",
		Type:            "scheduled",
		Title:           "【予定】明日のゲーム配信",
		Description:     pgtype.Text{String: "明日の配信予定", Valid: true},
		StartAt:         pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true},
		EndAt:           pgtype.Timestamptz{Valid: false},
		PublishedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Url:             "https://www.youtube.com/watch?v=scheduled456",
		ImageUrl:        pgtype.Text{String: "https://example.com/scheduled.jpg", Valid: true},
		Metrics:         []byte(`{"waiting": 89}`),
	})
	if err != nil {
		return fmt.Errorf("failed to upsert scheduled event: %w", err)
	}
	fmt.Printf("✅ Scheduled event created: %s\n", scheduledEvent.Title)

	// アーカイブ動画イベント
	videoEvent, err := queries.UpsertEvent(ctx, db.UpsertEventParams{
		PlatformID:      "youtube",
		SourceID:        source.ID,
		ExternalEventID: "video789",
		Type:            "video",
		Title:           "昨日のゲーム配信アーカイブ",
		Description:     pgtype.Text{String: "昨日の配信のアーカイブです", Valid: true},
		StartAt:         pgtype.Timestamptz{Valid: false}, // 動画は開始時刻なし
		EndAt:           pgtype.Timestamptz{Valid: false},
		PublishedAt:     pgtype.Timestamptz{Time: time.Now().Add(-24 * time.Hour), Valid: true},
		Url:             "https://www.youtube.com/watch?v=video789",
		ImageUrl:        pgtype.Text{String: "https://example.com/video.jpg", Valid: true},
		Metrics:         []byte(`{"views": 5678, "likes": 234, "comments": 45}`),
	})
	if err != nil {
		return fmt.Errorf("failed to upsert video event: %w", err)
	}
	fmt.Printf("✅ Video event created: %s\n", videoEvent.Title)

	// ============================================================================
	// 3. タイムライン取得
	// ============================================================================
	fmt.Println("\n=== 3. タイムライン取得 ===")

	timeline, err := queries.ListTimeline(ctx, db.ListTimelineParams{
		UserID:  1,
		Column2: pgtype.Timestamptz{Valid: false}, // before_time なし
		Limit:   50,
	})
	if err != nil {
		return fmt.Errorf("failed to list timeline: %w", err)
	}

	fmt.Printf("📺 Timeline items: %d\n\n", len(timeline))
	for i, item := range timeline {
		// 時刻表示
		timeStr := ""
		if item.StartAt.Valid {
			timeStr = item.StartAt.Time.Format("2006-01-02 15:04")
		} else if item.PublishedAt.Valid {
			timeStr = item.PublishedAt.Time.Format("2006-01-02 15:04")
		}

		// メトリクス表示
		metricsStr := ""
		if len(item.Metrics) > 0 {
			var metrics map[string]interface{}
			if err := json.Unmarshal(item.Metrics, &metrics); err == nil {
				metricsJSON, _ := json.Marshal(metrics)
				metricsStr = string(metricsJSON)
			}
		}

		fmt.Printf("%d. [%s] %s\n", i+1, item.Type, item.Title)
		fmt.Printf("   Source: %s (@%s)\n", item.SourceDisplayName.String, item.SourceHandle.String)
		fmt.Printf("   Time: %s\n", timeStr)
		fmt.Printf("   URL: %s\n", item.Url)
		if metricsStr != "" {
			fmt.Printf("   Metrics: %s\n", metricsStr)
		}
		fmt.Println()
	}

	// ============================================================================
	// 4. 配信中のイベント取得
	// ============================================================================
	fmt.Println("=== 4. 配信中のイベント ===")

	liveEvents, err := queries.ListLiveEvents(ctx, db.ListLiveEventsParams{
		UserID: 1,
		Limit:  10,
	})
	if err != nil {
		return fmt.Errorf("failed to list live events: %w", err)
	}

	fmt.Printf("🔴 Live events: %d\n\n", len(liveEvents))
	for _, item := range liveEvents {
		fmt.Printf("- %s by %s\n", item.Title, item.SourceDisplayName.String)
		fmt.Printf("  Started: %s\n", item.StartAt.Time.Format("15:04"))
	}

	// ============================================================================
	// 5. 今後の予定イベント取得
	// ============================================================================
	fmt.Println("\n=== 5. 今後の予定イベント ===")

	upcomingEvents, err := queries.ListUpcomingEvents(ctx, db.ListUpcomingEventsParams{
		UserID: 1,
		Limit:  10,
	})
	if err != nil {
		return fmt.Errorf("failed to list upcoming events: %w", err)
	}

	fmt.Printf("📅 Upcoming events: %d\n\n", len(upcomingEvents))
	for _, item := range upcomingEvents {
		fmt.Printf("- %s by %s\n", item.Title, item.SourceDisplayName.String)
		fmt.Printf("  Scheduled: %s\n", item.StartAt.Time.Format("2006-01-02 15:04"))
	}

	// ============================================================================
	// 6. 購読一覧取得
	// ============================================================================
	fmt.Println("\n=== 6. 購読一覧 ===")

	subscriptions, err := queries.ListUserEnabledSubscriptions(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to list subscriptions: %w", err)
	}

	fmt.Printf("⭐ Subscriptions: %d\n\n", len(subscriptions))
	for _, sub := range subscriptions {
		fmt.Printf("- %s (@%s) [%s]\n",
			sub.DisplayName.String,
			sub.Handle.String,
			sub.PlatformID)
		fmt.Printf("  Status: %s, Priority: %d\n", sub.FetchStatus, sub.Priority)
		fmt.Printf("  Subscribed: %s\n", sub.SubscribedAt.Time.Format("2006-01-02"))
	}

	return nil
}

// RunExample は例を実行
func RunExample() {
	dbURL := "postgresql://user:pass@localhost:26257/pixicast?sslmode=disable"

	if err := TimelineExample(dbURL); err != nil {
		log.Fatalf("Example failed: %v", err)
	}

	fmt.Println("\n✅ Example completed successfully!")
}

