-- query_timeline.sql
-- Timeline events（タイムライン項目）に関するクエリ

-- ============================================================================
-- UpsertEvent: イベント情報のupsert
-- ============================================================================
-- name: UpsertEvent :one
INSERT INTO events (
    platform_id,
    source_id,
    external_event_id,
    type,
    title,
    description,
    start_at,
    end_at,
    published_at,
    url,
    image_url,
    metrics,
    duration,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, now()
)
ON CONFLICT (platform_id, external_event_id)
DO UPDATE SET
    type = EXCLUDED.type,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    start_at = EXCLUDED.start_at,
    end_at = EXCLUDED.end_at,
    published_at = EXCLUDED.published_at,
    url = EXCLUDED.url,
    image_url = EXCLUDED.image_url,
    metrics = EXCLUDED.metrics,
    duration = EXCLUDED.duration,
    updated_at = now()
RETURNING *;

-- ============================================================================
-- GetEventByID: IDでイベント取得
-- ============================================================================
-- name: GetEventByID :one
SELECT * FROM events
WHERE id = $1;

-- ============================================================================
-- GetEventByExternalID: 外部IDでイベント取得
-- ============================================================================
-- name: GetEventByExternalID :one
SELECT * FROM events
WHERE platform_id = $1 AND external_event_id = $2;

-- ============================================================================
-- ListTimeline: ユーザーのタイムラインを取得
-- ============================================================================
-- name: ListTimeline :many
SELECT 
    e.id,
    e.platform_id,
    e.source_id,
    e.external_event_id,
    e.type,
    e.title,
    e.description,
    e.start_at,
    e.end_at,
    e.published_at,
    e.url,
    e.image_url,
    e.metrics,
    e.duration,
    e.created_at,
    e.updated_at,
    s.display_name as source_display_name,
    s.thumbnail_url as source_thumbnail_url,
    s.handle as source_handle,
    s.external_id as source_external_id
FROM events e
JOIN sources s ON e.source_id = s.id
JOIN user_subscriptions us ON s.id = us.source_id
WHERE 
    us.user_id = $1
    AND us.enabled = true
    AND (
        $2::timestamptz IS NULL
        OR COALESCE(e.start_at, e.published_at) < $2
    )
ORDER BY COALESCE(e.start_at, e.published_at) DESC NULLS LAST
LIMIT $3;

-- ============================================================================
-- ListTimelineBySource: 特定ソースのタイムラインを取得
-- ============================================================================
-- name: ListTimelineBySource :many
SELECT * FROM events
WHERE source_id = $1
ORDER BY COALESCE(start_at, published_at) DESC NULLS LAST
LIMIT $2;

-- ============================================================================
-- ListLiveEvents: 配信中のイベントを取得
-- ============================================================================
-- name: ListLiveEvents :many
SELECT 
    e.id,
    e.platform_id,
    e.source_id,
    e.external_event_id,
    e.type,
    e.title,
    e.description,
    e.start_at,
    e.end_at,
    e.published_at,
    e.url,
    e.image_url,
    e.metrics,
    e.created_at,
    e.updated_at,
    s.display_name as source_display_name,
    s.thumbnail_url as source_thumbnail_url,
    s.handle as source_handle
FROM events e
JOIN sources s ON e.source_id = s.id
JOIN user_subscriptions us ON s.id = us.source_id
WHERE 
    us.user_id = $1
    AND us.enabled = true
    AND e.type = 'live'
    AND e.start_at IS NOT NULL
    AND e.start_at <= now()
    AND (e.end_at IS NULL OR e.end_at > now())
ORDER BY e.start_at DESC
LIMIT $2;

-- ============================================================================
-- ListUpcomingEvents: 今後予定されているイベントを取得
-- ============================================================================
-- name: ListUpcomingEvents :many
SELECT 
    e.id,
    e.platform_id,
    e.source_id,
    e.external_event_id,
    e.type,
    e.title,
    e.description,
    e.start_at,
    e.end_at,
    e.published_at,
    e.url,
    e.image_url,
    e.metrics,
    e.created_at,
    e.updated_at,
    s.display_name as source_display_name,
    s.thumbnail_url as source_thumbnail_url,
    s.handle as source_handle
FROM events e
JOIN sources s ON e.source_id = s.id
JOIN user_subscriptions us ON s.id = us.source_id
WHERE 
    us.user_id = $1
    AND us.enabled = true
    AND e.type IN ('scheduled', 'premiere')
    AND e.start_at IS NOT NULL
    AND e.start_at > now()
ORDER BY e.start_at ASC
LIMIT $2;

-- ============================================================================
-- ListEventsByType: タイプ別にイベントを取得
-- ============================================================================
-- name: ListEventsByType :many
SELECT 
    e.id,
    e.platform_id,
    e.source_id,
    e.external_event_id,
    e.type,
    e.title,
    e.description,
    e.start_at,
    e.end_at,
    e.published_at,
    e.url,
    e.image_url,
    e.metrics,
    e.created_at,
    e.updated_at,
    s.display_name as source_display_name,
    s.thumbnail_url as source_thumbnail_url,
    s.handle as source_handle
FROM events e
JOIN sources s ON e.source_id = s.id
JOIN user_subscriptions us ON s.id = us.source_id
WHERE 
    us.user_id = $1
    AND us.enabled = true
    AND e.type = $2
ORDER BY COALESCE(e.start_at, e.published_at) DESC NULLS LAST
LIMIT $3;

-- ============================================================================
-- DeleteOldEvents: 古いイベントを削除（アーカイブ）
-- ============================================================================
-- name: DeleteOldEvents :exec
DELETE FROM events
WHERE 
    type = 'video'
    AND published_at < now() - INTERVAL '90 days';

-- ============================================================================
-- CountEventsBySource: ソース別のイベント数をカウント
-- ============================================================================
-- name: CountEventsBySource :one
SELECT COUNT(*) FROM events
WHERE source_id = $1;

