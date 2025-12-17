package ingest

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kinchoKayaba/pixicast/backend/db"
	"github.com/kinchoKayaba/pixicast/backend/internal/youtube"
	ytapi "google.golang.org/api/youtube/v3"
)

// FetchAndSaveChannelVideos はチャンネルの動画を取得してDBに保存
func FetchAndSaveChannelVideos(
	ctx context.Context,
	queries *db.Queries,
	youtubeClient *youtube.Client,
	sourceID pgtype.UUID,
	channelID string,
	maxResults int64,
) error {
	return FetchAndSaveChannelVideosSince(ctx, queries, youtubeClient, sourceID, channelID, maxResults, "")
}

// FetchAndSaveChannelVideosSince は指定日時以降のチャンネル動画を取得してDBに保存
func FetchAndSaveChannelVideosSince(
	ctx context.Context,
	queries *db.Queries,
	youtubeClient *youtube.Client,
	sourceID pgtype.UUID,
	channelID string,
	maxResults int64,
	publishedAfter string,
) error {
	if publishedAfter != "" {
		log.Printf("Fetching videos for channel: %s (since %s)", channelID, publishedAfter)
	} else {
		log.Printf("Fetching videos for channel: %s", channelID)
	}

	// 動画を取得
	videos, err := youtubeClient.GetChannelVideosSince(ctx, channelID, maxResults, publishedAfter)
	if err != nil {
		return fmt.Errorf("failed to get videos: %w", err)
	}

	// 動画の詳細情報をバッチで取得
	var videoIDs []string
	for _, video := range videos {
		if video.Id != nil && video.Id.VideoId != "" {
			videoIDs = append(videoIDs, video.Id.VideoId)
		}
	}

	videoDetails, err := youtubeClient.GetVideosDetails(ctx, videoIDs)
	if err != nil {
		return fmt.Errorf("failed to get video details: %w", err)
	}

	// 動画IDをキーとしたマップを作成
	detailsMap := make(map[string]*ytapi.Video)
	for _, detail := range videoDetails {
		detailsMap[detail.Id] = detail
	}

	// 各動画をDBに保存
	savedCount := 0
	skippedCount := 0
	for _, video := range videos {
		detail, ok := detailsMap[video.Id.VideoId]
		if !ok {
			log.Printf("⚠️ Video details not found for: %s (will save with empty duration/stats)", video.Id.VideoId)
			skippedCount++
			// 詳細情報がなくても基本情報は保存する
		}

		// サムネイルURL
		thumbnailUrl := ""
		if video.Snippet.Thumbnails != nil && video.Snippet.Thumbnails.High != nil {
			thumbnailUrl = video.Snippet.Thumbnails.High.Url
		}

		// published_atをパース
		publishedAt, err := time.Parse(time.RFC3339, video.Snippet.PublishedAt)
		if err != nil {
			log.Printf("Failed to parse published_at: %v", err)
			publishedAt = time.Now()
		}

		// 再生回数（詳細情報がない場合は0）
		viewCount := int64(0)
		if detail != nil && detail.Statistics != nil {
			viewCount = int64(detail.Statistics.ViewCount)
		}

		// 動画の長さ（詳細情報がない場合は空）
		duration := ""
		if detail != nil && detail.ContentDetails != nil && detail.ContentDetails.Duration != "" {
			duration = parseDuration(detail.ContentDetails.Duration)
		}

		// イベントタイプを判定
		eventType := "video"
		if video.Snippet.LiveBroadcastContent == "live" {
			eventType = "live"
		} else if video.Snippet.LiveBroadcastContent == "upcoming" {
			eventType = "scheduled"
		}

		// metricsをJSON形式で保存
		var metrics []byte
		if viewCount > 0 {
			metrics = []byte(fmt.Sprintf(`{"views": %d}`, viewCount))
		}

		// DBに保存
		_, err = queries.UpsertEvent(ctx, db.UpsertEventParams{
			PlatformID:      "youtube",
			SourceID:        sourceID,
			ExternalEventID: video.Id.VideoId,
			Type:            eventType,
			Title:           video.Snippet.Title,
			Description:     pgtype.Text{String: video.Snippet.Description, Valid: true},
			StartAt:         pgtype.Timestamptz{},
			EndAt:           pgtype.Timestamptz{},
			PublishedAt:     pgtype.Timestamptz{Time: publishedAt, Valid: true},
			Url:             fmt.Sprintf("https://www.youtube.com/watch?v=%s", video.Id.VideoId),
			ImageUrl:        pgtype.Text{String: thumbnailUrl, Valid: thumbnailUrl != ""},
			Metrics:         metrics,
			Duration:        pgtype.Text{String: duration, Valid: duration != ""},
		})
		if err != nil {
			log.Printf("Failed to upsert event %s: %v", video.Id.VideoId, err)
			continue
		}

		savedCount++
	}

	if skippedCount > 0 {
		log.Printf("✅ Saved %d videos for channel: %s (%d videos without details)", savedCount, channelID, skippedCount)
	} else {
		log.Printf("✅ Saved %d videos for channel: %s", savedCount, channelID)
	}
	return nil
}

// parseDuration はISO 8601形式の動画時間をHH:MM:SSまたはMM:SS形式に変換
func parseDuration(isoDuration string) string {
	if isoDuration == "" {
		return ""
	}

	// 正規表現でISO 8601形式をパース: PT(数字H)?(数字M)?(数字S)?
	re := regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)
	matches := re.FindStringSubmatch(isoDuration)
	
	if matches == nil {
		log.Printf("⚠️ Failed to parse duration: %s", isoDuration)
		return ""
	}

	hours := 0
	minutes := 0
	seconds := 0

	if matches[1] != "" {
		hours, _ = strconv.Atoi(matches[1])
	}
	if matches[2] != "" {
		minutes, _ = strconv.Atoi(matches[2])
	}
	if matches[3] != "" {
		seconds, _ = strconv.Atoi(matches[3])
	}

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
