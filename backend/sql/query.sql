-- name: ListPrograms :many
SELECT * FROM programs
ORDER BY start_at ASC;

-- name: CreateProgram :one
INSERT INTO programs (
  title, start_at, end_at, platform_name, image_url, link_url
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpsertSource :one
INSERT INTO sources (
  platform_id, external_id, handle, display_name, thumbnail_url, uploads_playlist_id, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, now()
)
ON CONFLICT (platform_id, external_id)
DO UPDATE SET
  handle = EXCLUDED.handle,
  display_name = EXCLUDED.display_name,
  thumbnail_url = EXCLUDED.thumbnail_url,
  uploads_playlist_id = EXCLUDED.uploads_playlist_id,
  updated_at = now()
RETURNING *;

-- name: GetSourceByID :one
SELECT * FROM sources
WHERE id = $1;

-- name: GetSourceByExternalID :one
SELECT * FROM sources
WHERE platform_id = $1 AND external_id = $2;

-- name: UpsertUserSubscription :one
INSERT INTO user_subscriptions (
  user_id, source_id, enabled, updated_at
) VALUES (
  $1, $2, $3, now()
)
ON CONFLICT (user_id, source_id)
DO UPDATE SET
  enabled = EXCLUDED.enabled,
  updated_at = now()
RETURNING *;

-- name: GetUserSubscription :one
SELECT * FROM user_subscriptions
WHERE user_id = $1 AND source_id = $2;

-- name: ListUserSubscriptions :many
SELECT s.*, us.enabled, us.created_at as subscribed_at
FROM user_subscriptions us
JOIN sources s ON us.source_id = s.id
WHERE us.user_id = $1 AND us.enabled = true
ORDER BY us.created_at DESC;
