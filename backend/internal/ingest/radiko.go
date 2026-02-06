package ingest

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kinchoKayaba/pixicast/backend/db"
	"github.com/kinchoKayaba/pixicast/backend/internal/radiko"
)

// FetchAndSaveRadikoPrograms は指定局のRadiko番組を取得してDBに保存
func FetchAndSaveRadikoPrograms(
	ctx context.Context,
	queries *db.Queries,
	radikoClient *radiko.Client,
	sourceID pgtype.UUID,
	stationID string,
	since string,
) error {
	log.Printf("📻 [Radiko] Fetching programs for station: %s (since %s)", stationID, since)

	// 週間番組表を取得
	programs, err := radikoClient.GetWeeklyPrograms(ctx, stationID)
	if err != nil {
		return fmt.Errorf("failed to get weekly programs: %w", err)
	}

	// sinceの時刻を解析
	sinceTime, err := time.Parse(time.RFC3339, since)
	if err != nil {
		return fmt.Errorf("failed to parse since time: %w", err)
	}

	savedCount := 0
	for _, prog := range programs {
		// since以降の番組のみ保存
		if prog.StartTime.Before(sinceTime) {
			continue
		}

		// イベントをDBに保存
		err := queries.UpsertEvent(ctx, db.UpsertEventParams{
			PlatformID:      "radiko",
			SourceID:        sourceID,
			ExternalEventID: prog.ID,
			Type:            "radio", // 新しいイベントタイプ
			Title:           prog.Title,
			Description: pgtype.Text{
				String: prog.Description,
				Valid:  prog.Description != "",
			},
			StartAt: pgtype.Timestamptz{
				Time:  prog.StartTime,
				Valid: true,
			},
			EndAt: pgtype.Timestamptz{
				Time:  prog.EndTime,
				Valid: true,
			},
			PublishedAt: pgtype.Timestamptz{
				Time:  prog.StartTime,
				Valid: true,
			},
			Url: prog.URL,
			ImageUrl: pgtype.Text{
				String: prog.ImageURL,
				Valid:  prog.ImageURL != "",
			},
			Metrics: nil, // Radikoには視聴数等の統計情報がない
			Duration: pgtype.Text{
				String: formatDuration(prog.Duration),
				Valid:  true,
			},
		})

		if err != nil {
			log.Printf("⚠️  Failed to save program %s: %v", prog.Title, err)
			continue
		}

		savedCount++
	}

	log.Printf("✅ [Radiko] Saved %d/%d programs for station %s", savedCount, len(programs), stationID)
	return nil
}

// FetchAndSaveRadikoStation はラジオ局情報を取得してDBに保存
func FetchAndSaveRadikoStation(
	ctx context.Context,
	queries *db.Queries,
	radikoClient *radiko.Client,
	stationID string,
	areaID string,
) (pgtype.UUID, error) {
	log.Printf("📻 [Radiko] Fetching station info: %s (area: %s)", stationID, areaID)

	// エリアの全局を取得
	stations, err := radikoClient.GetStations(ctx, areaID)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("failed to get stations: %w", err)
	}

	// 指定stationIDの局を探す
	var targetStation *radiko.Station
	for _, station := range stations {
		if station.ID == stationID {
			targetStation = &station
			break
		}
	}

	if targetStation == nil {
		return pgtype.UUID{}, fmt.Errorf("station not found: %s", stationID)
	}

	// sourcesテーブルにupsert
	source, err := queries.UpsertSource(ctx, db.UpsertSourceParams{
		PlatformID: "radiko",
		ExternalID: targetStation.ID,
		Handle: pgtype.Text{
			String: targetStation.ID,
			Valid:  true,
		},
		DisplayName: pgtype.Text{
			String: targetStation.Name,
			Valid:  true,
		},
		ThumbnailUrl: pgtype.Text{
			String: targetStation.LogoURL,
			Valid:  targetStation.LogoURL != "",
		},
		UploadsPlaylistID: pgtype.Text{}, // Radikoには該当なし
		ApplePodcastUrl:   pgtype.Text{}, // Radikoには該当なし
	})

	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("failed to upsert source: %w", err)
	}

	log.Printf("✅ [Radiko] Saved station: %s (%s)", targetStation.Name, source.ID.String())
	return source.ID, nil
}

// formatDuration は秒数を HH:MM:SS 形式に変換
func formatDuration(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}
