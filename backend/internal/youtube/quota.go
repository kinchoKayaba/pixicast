package youtube

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kinchoKayaba/pixicast/backend/db"
)

// QuotaCost はYouTube API各エンドポイントのQuotaコスト
var QuotaCost = map[string]int{
	"channels.list":      1,
	"videos.list":        1,
	"playlistItems.list": 1,
	"search.list":        100,
	"activities.list":    1,
}

// QuotaTracker はAPI Quota使用量を追跡
type QuotaTracker struct {
	queries      *db.Queries
	dailyUsed    atomic.Int32
	dailyLimit   int32
	lastResetDate string
}

// NewQuotaTracker は新しいQuotaTrackerを作成
func NewQuotaTracker(queries *db.Queries, dailyLimit int32) *QuotaTracker {
	tracker := &QuotaTracker{
		queries:    queries,
		dailyLimit: dailyLimit,
		lastResetDate: time.Now().Format("2006-01-02"),
	}

	// 起動時に当日の使用量を読み込み
	ctx := context.Background()
	now := time.Now()
	todayDate := pgtype.Date{Time: now, Valid: true}
	usage, err := queries.GetDailyAPIQuotaUsage(ctx, db.GetDailyAPIQuotaUsageParams{
		Date:       todayDate,
		PlatformID: "youtube",
	})
	if err == nil {
		totalUsed := toInt32(usage.TotalQuotaUsed)
		tracker.dailyUsed.Store(totalUsed)
		log.Printf("📊 YouTube API Quota today: %d/%d (%.1f%%)",
			totalUsed,
			dailyLimit,
			float64(totalUsed)/float64(dailyLimit)*100,
		)
	}

	return tracker
}

// RecordUsage はAPI使用量を記録
func (qt *QuotaTracker) RecordUsage(ctx context.Context, endpoint string, cost int) error {
	// 日付が変わったらリセット
	today := time.Now().Format("2006-01-02")
	if qt.lastResetDate != today {
		qt.dailyUsed.Store(0)
		qt.lastResetDate = today
		log.Println("🔄 YouTube API Quota reset (new day)")
	}

	// 使用量を加算
	newUsed := qt.dailyUsed.Add(int32(cost))

	// データベースに記録
	todayDate := pgtype.Date{Time: time.Now(), Valid: true}
	err := qt.queries.RecordAPIQuotaUsage(ctx, db.RecordAPIQuotaUsageParams{
		Date:       todayDate,
		PlatformID: "youtube",
		Endpoint:   endpoint,
		QuotaCost:  int32(cost),
	})

	if err != nil {
		log.Printf("⚠️  Failed to record API quota usage: %v", err)
	}

	// 警告レベルをチェック
	usagePercent := float64(newUsed) / float64(qt.dailyLimit) * 100
	if usagePercent >= 90 {
		log.Printf("🚨 WARNING: YouTube API Quota at %.1f%% (%d/%d)",
			usagePercent, newUsed, qt.dailyLimit)
	} else if usagePercent >= 75 {
		log.Printf("⚠️  YouTube API Quota at %.1f%% (%d/%d)",
			usagePercent, newUsed, qt.dailyLimit)
	}

	return err
}

// GetUsage は現在の使用量を取得
func (qt *QuotaTracker) GetUsage() int32 {
	return qt.dailyUsed.Load()
}

// GetRemaining は残りのQuotaを取得
func (qt *QuotaTracker) GetRemaining() int32 {
	used := qt.dailyUsed.Load()
	remaining := qt.dailyLimit - used
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetUsagePercent は使用率を取得
func (qt *QuotaTracker) GetUsagePercent() float64 {
	used := qt.dailyUsed.Load()
	return float64(used) / float64(qt.dailyLimit) * 100
}

// CanUse は指定されたコストのAPI呼び出しが可能かチェック
func (qt *QuotaTracker) CanUse(cost int) bool {
	used := qt.dailyUsed.Load()
	return used+int32(cost) <= qt.dailyLimit
}

// toInt32 はinterface{}からint32に変換
func toInt32(v interface{}) int32 {
	switch n := v.(type) {
	case int64:
		return int32(n)
	case int32:
		return n
	case float64:
		return int32(n)
	default:
		return 0
	}
}
