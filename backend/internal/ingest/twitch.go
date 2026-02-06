package ingest

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kinchoKayaba/pixicast/backend/db"
	"github.com/kinchoKayaba/pixicast/backend/internal/twitch"
)

func FetchAndSaveTwitchVideosSince(
	ctx context.Context,
	queries *db.Queries,
	twitchClient *twitch.Client,
	sourceID pgtype.UUID,
	userID string,
	publishedAfter string,
) error {
	log.Printf("Fetching Twitch content (live streams + videos) for user: %s (since %s)", userID, publishedAfter)

	var cutoff time.Time
	if publishedAfter != "" {
		cutoff, _ = time.Parse(time.RFC3339, publishedAfter)
	}

	savedCount := 0

	// 1. まず現在配信中のライブストリームを取得
	var liveStreamStartTimes []time.Time // ライブストリームの開始時刻リスト
	var currentLiveStreamIDs []string     // 現在配信中のstream IDリスト
	streams, err := twitchClient.GetStreams(ctx, userID)
	if err != nil {
		log.Printf("⚠️ Failed to get live streams (non-fatal): %v", err)
	} else {
		for _, stream := range streams {
			currentLiveStreamIDs = append(currentLiveStreamIDs, stream.ID)
			// ライブ配信用のサムネイル
			thumbnailURL := strings.ReplaceAll(stream.ThumbnailURL, "{width}", "640")
			thumbnailURL = strings.ReplaceAll(thumbnailURL, "{height}", "360")

			// ライブ配信のURL
			liveURL := fmt.Sprintf("https://www.twitch.tv/%s", stream.UserLogin)

			metrics := []byte(fmt.Sprintf(`{"viewers": %d}`, stream.ViewerCount))

			_, err = queries.UpsertEvent(ctx, db.UpsertEventParams{
				PlatformID:      "twitch",
				SourceID:        sourceID,
				ExternalEventID: stream.ID,
				Type:            "live",
				Title:           stream.Title,
				Description:     pgtype.Text{String: fmt.Sprintf("🔴 LIVE - %s", stream.GameName), Valid: true},
				StartAt:         pgtype.Timestamptz{Time: stream.StartedAt, Valid: true},
				EndAt:           pgtype.Timestamptz{},
				PublishedAt:     pgtype.Timestamptz{Time: stream.StartedAt, Valid: true},
				Url:             liveURL,
				ImageUrl:        pgtype.Text{String: thumbnailURL, Valid: thumbnailURL != ""},
				Metrics:         metrics,
				Duration:        pgtype.Text{String: "", Valid: false},
			})
			if err != nil {
				log.Printf("Failed to upsert live stream %s: %v", stream.ID, err)
				continue
			}
			log.Printf("✅ Saved LIVE stream: %s (%d viewers)", stream.Title, stream.ViewerCount)
			
			// ライブストリームの開始時刻を記録（VOD重複除外用）
			liveStreamStartTimes = append(liveStreamStartTimes, stream.StartedAt)
			savedCount++
		}
	}

	// 2. 過去の配信動画（VOD）を取得
	videos, err := twitchClient.GetVideos(ctx, userID, 100)
	if err != nil {
		return fmt.Errorf("failed to get videos: %w", err)
	}

	for _, video := range videos {
		if !cutoff.IsZero() && video.CreatedAt.Before(cutoff) {
			continue
		}

		// 配信中のライブストリームと重複しているVODは除外
		// （VODの作成時刻が、いずれかのライブストリーム開始時刻の6時間以内）
		// ※ Twitchでは配信中にタイトルが変わることがあるため、時間範囲を広めに設定
		isDuplicate := false
		for _, liveStartTime := range liveStreamStartTimes {
			timeDiff := video.CreatedAt.Sub(liveStartTime)
			if timeDiff.Abs() < 6*time.Hour {
				log.Printf("⏭️  Skipping VOD (duplicate of live stream): %s", video.Title)
				isDuplicate = true
				break
			}
		}
		if isDuplicate {
			continue
		}

		// VODは常に"video"タイプとして保存（配信終了後のアーカイブのため）
		eventType := "video"

		metrics := []byte(fmt.Sprintf(`{"views": %d}`, video.ViewCount))

		// TwitchのサムネイルURLのプレースホルダーを実際のサイズに置換
		thumbnailURL := strings.ReplaceAll(video.ThumbnailURL, "%{width}", "640")
		thumbnailURL = strings.ReplaceAll(thumbnailURL, "%{height}", "360")

		_, err = queries.UpsertEvent(ctx, db.UpsertEventParams{
			PlatformID:      "twitch",
			SourceID:        sourceID,
			ExternalEventID: video.ID,
			Type:            eventType,
			Title:           video.Title,
			Description:     pgtype.Text{String: video.Description, Valid: true},
			StartAt:         pgtype.Timestamptz{},
			EndAt:           pgtype.Timestamptz{},
			PublishedAt:     pgtype.Timestamptz{Time: video.CreatedAt, Valid: true},
			Url:             video.URL,
			ImageUrl:        pgtype.Text{String: thumbnailURL, Valid: thumbnailURL != ""},
			Metrics:         metrics,
			Duration:        pgtype.Text{String: video.Duration, Valid: video.Duration != ""},
		})
		if err != nil {
			log.Printf("Failed to upsert event %s: %v", video.ID, err)
			continue
		}
		savedCount++
	}

	log.Printf("✅ Saved %d Twitch content items (live streams + videos) for user: %s", savedCount, userID)
	return nil
}

