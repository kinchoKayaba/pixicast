package ingest

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kinchoKayaba/pixicast/backend/db"
	"github.com/kinchoKayaba/pixicast/backend/internal/podcast"
)

func FetchAndSavePodcastEpisodesSince(
	ctx context.Context,
	queries *db.Queries,
	podcastClient *podcast.Client,
	sourceID pgtype.UUID,
	feedURL string,
	publishedAfter string,
) error {
	log.Printf("Fetching podcast episodes from: %s (since %s)", feedURL, publishedAfter)

	// SourceからApple Podcasts URLを取得
	source, err := queries.GetSourceByID(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get source: %w", err)
	}
	
	applePodcastURL := ""
	if source.ApplePodcastUrl.Valid {
		applePodcastURL = source.ApplePodcastUrl.String
	}

	_, episodes, err := podcastClient.ParseFeed(ctx, feedURL)
	if err != nil {
		return fmt.Errorf("failed to parse feed: %w", err)
	}

	var cutoff time.Time
	if publishedAfter != "" {
		cutoff, _ = time.Parse(time.RFC3339, publishedAfter)
	}

	savedCount := 0
	for _, episode := range episodes {
		if !cutoff.IsZero() && episode.PublishedAt.Before(cutoff) {
			continue
		}

		// エピソードURLを決定: Apple Podcasts URL > episode.URL
		episodeURL := episode.URL
		if applePodcastURL != "" {
			episodeURL = applePodcastURL // Apple Podcasts番組ページを優先
		}

		_, err = queries.UpsertEvent(ctx, db.UpsertEventParams{
			PlatformID:      "podcast",
			SourceID:        sourceID,
			ExternalEventID: episode.GUID,
			Type:            "episode",
			Title:           episode.Title,
			Description:     pgtype.Text{String: episode.Description, Valid: true},
			StartAt:         pgtype.Timestamptz{},
			EndAt:           pgtype.Timestamptz{},
			PublishedAt:     pgtype.Timestamptz{Time: episode.PublishedAt, Valid: true},
			Url:             episodeURL, // Apple Podcasts URLまたはフォールバック
			ImageUrl:        pgtype.Text{String: episode.ImageURL, Valid: episode.ImageURL != ""},
			Metrics:         nil,
			Duration:        pgtype.Text{String: episode.Duration, Valid: episode.Duration != ""},
		})
		if err != nil {
			log.Printf("Failed to upsert episode %s: %v", episode.GUID, err)
			continue
		}
		savedCount++
	}

	log.Printf("✅ Saved %d podcast episodes from: %s", savedCount, feedURL)
	return nil
}

