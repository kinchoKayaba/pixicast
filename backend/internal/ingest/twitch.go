package ingest

import (
	"context"
	"fmt"
	"log"
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
	log.Printf("Fetching Twitch videos for user: %s (since %s)", userID, publishedAfter)

	videos, err := twitchClient.GetVideos(ctx, userID, 100)
	if err != nil {
		return fmt.Errorf("failed to get videos: %w", err)
	}

	var cutoff time.Time
	if publishedAfter != "" {
		cutoff, _ = time.Parse(time.RFC3339, publishedAfter)
	}

	savedCount := 0
	for _, video := range videos {
		if !cutoff.IsZero() && video.CreatedAt.Before(cutoff) {
			continue
		}

		eventType := "video"
		if video.Type == "live" {
			eventType = "live"
		}

		metrics := []byte(fmt.Sprintf(`{"views": %d}`, video.ViewCount))

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
			ImageUrl:        pgtype.Text{String: video.ThumbnailURL, Valid: true},
			Metrics:         metrics,
			Duration:        pgtype.Text{String: video.Duration, Valid: video.Duration != ""},
		})
		if err != nil {
			log.Printf("Failed to upsert event %s: %v", video.ID, err)
			continue
		}
		savedCount++
	}

	log.Printf("✅ Saved %d Twitch videos for user: %s", savedCount, userID)
	return nil
}

