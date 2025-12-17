-- query_subscriptions.sql
-- User subscriptions（ユーザーの購読情報）に関するクエリ

-- ============================================================================
-- UpsertUserSubscription: 購読情報のupsert
-- ============================================================================
-- name: UpsertUserSubscription :one
INSERT INTO user_subscriptions (
    user_id,
    source_id,
    enabled,
    priority,
    updated_at
) VALUES (
    $1, $2, $3, $4, now()
)
ON CONFLICT (user_id, source_id)
DO UPDATE SET
    enabled = EXCLUDED.enabled,
    priority = EXCLUDED.priority,
    updated_at = now()
RETURNING *;

-- ============================================================================
-- GetUserSubscription: 特定の購読情報を取得
-- ============================================================================
-- name: GetUserSubscription :one
SELECT * FROM user_subscriptions
WHERE user_id = $1 AND source_id = $2;

-- ============================================================================
-- ListUserSubscriptions: ユーザーの購読一覧を取得（source情報付き）
-- ============================================================================
-- name: ListUserSubscriptions :many
SELECT 
    s.id,
    s.platform_id,
    s.external_id,
    s.handle,
    s.display_name,
    s.thumbnail_url,
    s.uploads_playlist_id,
    s.fetch_status,
    s.last_fetched_at,
    s.created_at,
    s.updated_at,
    us.enabled,
    us.priority,
    us.created_at as subscribed_at
FROM user_subscriptions us
JOIN sources s ON us.source_id = s.id
WHERE us.user_id = $1
ORDER BY us.priority DESC, us.created_at DESC;

-- ============================================================================
-- ListUserEnabledSubscriptions: ユーザーの有効な購読一覧を取得
-- ============================================================================
-- name: ListUserEnabledSubscriptions :many
SELECT 
    s.id,
    s.platform_id,
    s.external_id,
    s.handle,
    s.display_name,
    s.thumbnail_url,
    s.uploads_playlist_id,
    s.fetch_status,
    s.last_fetched_at,
    s.created_at,
    s.updated_at,
    us.enabled,
    us.is_favorite,
    us.priority,
    us.created_at as subscribed_at
FROM user_subscriptions us
JOIN sources s ON us.source_id = s.id
WHERE us.user_id = $1 AND us.enabled = true
ORDER BY us.priority DESC, us.created_at DESC;

-- ============================================================================
-- UpdateSubscriptionEnabled: 購読の有効/無効を切り替え
-- ============================================================================
-- name: UpdateSubscriptionEnabled :one
UPDATE user_subscriptions
SET
    enabled = $3,
    updated_at = now()
WHERE user_id = $1 AND source_id = $2
RETURNING *;

-- ============================================================================
-- UpdateSubscriptionPriority: 購読の優先度を更新
-- ============================================================================
-- name: UpdateSubscriptionPriority :one
UPDATE user_subscriptions
SET
    priority = $3,
    updated_at = now()
WHERE user_id = $1 AND source_id = $2
RETURNING *;

-- ============================================================================
-- DeleteUserSubscription: 購読を削除
-- ============================================================================
-- name: DeleteUserSubscription :exec
DELETE FROM user_subscriptions
WHERE user_id = $1 AND source_id = $2;

-- ============================================================================
-- CountUserSubscriptions: ユーザーの購読数をカウント
-- ============================================================================
-- name: CountUserSubscriptions :one
SELECT COUNT(*) FROM user_subscriptions
WHERE user_id = $1 AND enabled = true;

-- ============================================================================
-- ListSubscribedSourceIDs: ユーザーが購読中のsource_idリストを取得
-- ============================================================================
-- name: ListSubscribedSourceIDs :many
SELECT source_id FROM user_subscriptions
WHERE user_id = $1 AND enabled = true
ORDER BY priority DESC, created_at DESC;

-- ============================================================================
-- ToggleSubscriptionFavorite: お気に入り状態を切り替え
-- ============================================================================
-- name: ToggleSubscriptionFavorite :one
UPDATE user_subscriptions
SET is_favorite = $3, updated_at = now()
WHERE user_id = $1 AND source_id = $2
RETURNING *;

-- ============================================================================
-- ListFavoriteSubscriptions: お気に入りの購読チャンネル一覧を取得
-- ============================================================================
-- name: ListFavoriteSubscriptions :many
SELECT
    s.id,
    s.platform_id,
    s.external_id,
    s.handle,
    s.display_name,
    s.thumbnail_url,
    s.uploads_playlist_id,
    s.last_fetched_at,
    s.fetch_status,
    s.created_at,
    s.updated_at,
    us.enabled,
    us.is_favorite,
    us.priority,
    us.created_at as subscribed_at
FROM user_subscriptions us
JOIN sources s ON us.source_id = s.id
WHERE us.user_id = $1 AND us.enabled = true AND us.is_favorite = true
ORDER BY us.priority DESC, us.created_at DESC;
