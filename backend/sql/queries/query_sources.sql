-- query_sources.sql
-- Sources（チャンネル/配信者）に関するクエリ

-- ============================================================================
-- UpsertSource: チャンネル情報のupsert
-- ============================================================================
-- name: UpsertSource :one
INSERT INTO sources (
    platform_id,
    external_id,
    handle,
    display_name,
    thumbnail_url,
    uploads_playlist_id,
    apple_podcast_url,
    fetch_status,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'ok', now()
)
ON CONFLICT (platform_id, external_id)
DO UPDATE SET
    handle = EXCLUDED.handle,
    display_name = EXCLUDED.display_name,
    thumbnail_url = EXCLUDED.thumbnail_url,
    uploads_playlist_id = EXCLUDED.uploads_playlist_id,
    apple_podcast_url = EXCLUDED.apple_podcast_url,
    fetch_status = EXCLUDED.fetch_status,
    updated_at = now()
RETURNING *;

-- ============================================================================
-- GetSourceByID: IDでソース取得
-- ============================================================================
-- name: GetSourceByID :one
SELECT * FROM sources
WHERE id = $1;

-- ============================================================================
-- GetSourceByExternalID: 外部ID（platform + external_id）でソース取得
-- ============================================================================
-- name: GetSourceByExternalID :one
SELECT * FROM sources
WHERE platform_id = $1 AND external_id = $2;

-- ============================================================================
-- ListSources: ソース一覧を取得
-- ============================================================================
-- name: ListSources :many
SELECT * FROM sources
ORDER BY created_at DESC
LIMIT $1;

-- ============================================================================
-- ListSourcesByPlatform: プラットフォーム別にソース一覧を取得
-- ============================================================================
-- name: ListSourcesByPlatform :many
SELECT * FROM sources
WHERE platform_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- ============================================================================
-- UpdateSourceFetchStatus: 取り込みステータスを更新
-- ============================================================================
-- name: UpdateSourceFetchStatus :one
UPDATE sources
SET
    fetch_status = $2,
    last_fetched_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- ============================================================================
-- ListSourcesForFetch: 取り込み対象のソースを取得
-- ============================================================================
-- name: ListSourcesForFetch :many
SELECT s.*
FROM sources s
WHERE 
    s.fetch_status = 'ok'
    AND (
        s.last_fetched_at IS NULL 
        OR s.last_fetched_at < now() - INTERVAL '10 minutes'
    )
ORDER BY s.last_fetched_at ASC NULLS FIRST
LIMIT $1;

-- ============================================================================
-- SearchSources: キーワードでソースを検索（全プラットフォーム）
-- ============================================================================
-- name: SearchSources :many
SELECT * FROM sources
WHERE
    (display_name ILIKE '%' || @query::TEXT || '%')
    OR (handle ILIKE '%' || @query::TEXT || '%')
ORDER BY display_name ASC
LIMIT @max_results;

-- ============================================================================
-- SearchSourcesByPlatform: プラットフォーム指定でキーワード検索
-- ============================================================================
-- name: SearchSourcesByPlatform :many
SELECT * FROM sources
WHERE
    platform_id = @platform_id
    AND (
        (display_name ILIKE '%' || @query::TEXT || '%')
        OR (handle ILIKE '%' || @query::TEXT || '%')
    )
ORDER BY display_name ASC
LIMIT @max_results;

-- ============================================================================
-- PopularSources: Pixicast内の人気チャンネル（購読者数順）
-- ============================================================================
-- name: PopularSources :many
SELECT s.*, COUNT(us.user_id) as subscriber_count
FROM sources s
JOIN user_subscriptions us ON s.id = us.source_id
WHERE us.enabled = true
GROUP BY s.id
ORDER BY subscriber_count DESC
LIMIT @max_results;

