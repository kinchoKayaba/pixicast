-- name: CalculateSourcePriority :exec
-- 全チャンネルの優先度を計算してsource_priorityテーブルを更新
WITH channel_popularity AS (
    SELECT
        us.source_id,
        COUNT(DISTINCT us.user_id) as subscriber_count,
        MAX(us.last_accessed_at) as last_access
    FROM user_subscriptions us
    WHERE us.enabled = true
    GROUP BY us.source_id
),
total_users AS (
    SELECT COUNT(DISTINCT user_id) as total
    FROM user_subscriptions
    WHERE enabled = true
),
calculated_priority AS (
    SELECT
        s.id as source_id,
        COALESCE(cp.subscriber_count, 0) as subscriber_count,
        CASE
            WHEN tu.total = 0 THEN 0.0
            ELSE CAST(COALESCE(cp.subscriber_count, 0) AS DECIMAL) / CAST(tu.total AS DECIMAL)
        END as popularity_ratio,
        CASE
            WHEN tu.total = 0 THEN 'low'
            WHEN CAST(COALESCE(cp.subscriber_count, 0) AS DECIMAL) / CAST(tu.total AS DECIMAL) >= 0.5 THEN 'high'
            WHEN CAST(COALESCE(cp.subscriber_count, 0) AS DECIMAL) / CAST(tu.total AS DECIMAL) >= 0.1 THEN 'medium'
            ELSE 'low'
        END as priority_level,
        CASE
            WHEN tu.total = 0 THEN 360
            WHEN CAST(COALESCE(cp.subscriber_count, 0) AS DECIMAL) / CAST(tu.total AS DECIMAL) >= 0.5 THEN 60
            WHEN CAST(COALESCE(cp.subscriber_count, 0) AS DECIMAL) / CAST(tu.total AS DECIMAL) >= 0.1 THEN 180
            ELSE 360
        END as update_interval_minutes
    FROM sources s
    CROSS JOIN total_users tu
    LEFT JOIN channel_popularity cp ON s.id = cp.source_id
    WHERE s.platform_id = 'youtube'
        AND s.fetch_status = 'ok'
)
INSERT INTO source_priority (
    source_id,
    subscriber_count,
    popularity_ratio,
    priority_level,
    update_interval_minutes,
    last_calculated_at,
    updated_at
)
SELECT
    source_id,
    subscriber_count,
    popularity_ratio,
    priority_level,
    update_interval_minutes,
    now(),
    now()
FROM calculated_priority
ON CONFLICT (source_id) DO UPDATE SET
    subscriber_count = EXCLUDED.subscriber_count,
    popularity_ratio = EXCLUDED.popularity_ratio,
    priority_level = EXCLUDED.priority_level,
    update_interval_minutes = EXCLUDED.update_interval_minutes,
    last_calculated_at = now(),
    updated_at = now();

-- name: GetSourcesByPriority :many
-- 優先度別にソースを取得
SELECT
    s.id,
    s.platform_id,
    s.external_id,
    s.display_name,
    s.uploads_playlist_id,
    s.last_fetched_at,
    sp.priority_level,
    sp.subscriber_count,
    sp.popularity_ratio,
    sp.update_interval_minutes
FROM sources s
JOIN source_priority sp ON s.id = sp.source_id
WHERE s.platform_id = $1
    AND s.fetch_status = 'ok'
    AND (
        s.last_fetched_at IS NULL
        OR s.last_fetched_at < now() - (sp.update_interval_minutes || ' minutes')::interval
    )
ORDER BY
    sp.priority_level DESC,
    s.last_fetched_at ASC NULLS FIRST
LIMIT $2;

-- name: GetHighPrioritySources :many
-- 高優先度チャンネルのみ取得
SELECT
    s.id,
    s.platform_id,
    s.external_id,
    s.display_name,
    s.uploads_playlist_id,
    s.last_fetched_at
FROM sources s
JOIN source_priority sp ON s.id = sp.source_id
WHERE sp.priority_level = 'high'
    AND s.fetch_status = 'ok'
    AND (
        s.last_fetched_at IS NULL
        OR s.last_fetched_at < now() - interval '60 minutes'
    )
ORDER BY s.last_fetched_at ASC NULLS FIRST
LIMIT $1;

-- name: RecordAPIQuotaUsage :exec
-- API使用量を記録
INSERT INTO api_quota_usage (
    date,
    platform_id,
    endpoint,
    quota_cost,
    request_count,
    created_at
)
VALUES ($1, $2, $3, $4, 1, now())
ON CONFLICT (date, platform_id, endpoint) DO UPDATE SET
    quota_cost = api_quota_usage.quota_cost + EXCLUDED.quota_cost,
    request_count = api_quota_usage.request_count + 1;

-- name: GetDailyAPIQuotaUsage :one
-- 日次API使用量を取得
SELECT
    COALESCE(SUM(quota_cost), 0) as total_quota_used,
    COALESCE(SUM(request_count), 0) as total_requests
FROM api_quota_usage
WHERE date = $1
    AND platform_id = $2;

-- name: GetAPIQuotaUsageByEndpoint :many
-- エンドポイント別のAPI使用量を取得
SELECT
    endpoint,
    SUM(quota_cost) as total_quota_cost,
    SUM(request_count) as total_requests
FROM api_quota_usage
WHERE date = $1
    AND platform_id = $2
GROUP BY endpoint
ORDER BY total_quota_cost DESC;

-- name: CreateUpdateSchedule :one
-- 更新スケジュールを作成
INSERT INTO update_schedule (
    source_id,
    scheduled_at,
    priority_level,
    status,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, 'pending', now(), now())
RETURNING *;

-- name: GetPendingSchedules :many
-- 実行待ちのスケジュールを取得
SELECT
    us.*,
    s.external_id,
    s.platform_id
FROM update_schedule us
JOIN sources s ON us.source_id = s.id
WHERE us.status = 'pending'
    AND us.scheduled_at <= now()
ORDER BY us.priority_level DESC, us.scheduled_at ASC
LIMIT $1;

-- name: UpdateScheduleStatus :exec
-- スケジュールステータスを更新
UPDATE update_schedule
SET
    status = $2,
    started_at = CASE WHEN $2 = 'running' THEN now() ELSE started_at END,
    completed_at = CASE WHEN $2 IN ('completed', 'failed') THEN now() ELSE completed_at END,
    error_message = $3,
    updated_at = now()
WHERE id = $1;

-- name: GetSourcePriorityStats :many
-- 優先度別の統計情報を取得
SELECT
    priority_level,
    COUNT(*) as source_count,
    AVG(subscriber_count) as avg_subscribers,
    AVG(popularity_ratio) as avg_popularity,
    update_interval_minutes
FROM source_priority
GROUP BY priority_level, update_interval_minutes
ORDER BY
    CASE priority_level
        WHEN 'high' THEN 1
        WHEN 'medium' THEN 2
        WHEN 'low' THEN 3
    END;
