-- query_users.sql

-- name: UpsertUser :one
INSERT INTO users (id, firebase_uid, plan_type, email, display_name, photo_url, is_anonymous, last_accessed_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
ON CONFLICT (firebase_uid) DO UPDATE SET
    email = COALESCE(EXCLUDED.email, users.email),
    display_name = COALESCE(EXCLUDED.display_name, users.display_name),
    photo_url = COALESCE(EXCLUDED.photo_url, users.photo_url),
    last_accessed_at = now(),
    updated_at = now()
RETURNING *;

-- name: GetUserByFirebaseUID :one
SELECT * FROM users WHERE firebase_uid = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUserPlan :one
UPDATE users SET plan_type = $2, is_anonymous = false, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: GetPlanLimit :one
SELECT * FROM plan_limits WHERE plan_type = $1;

-- name: ListAllPlanLimits :many
SELECT * FROM plan_limits ORDER BY price_monthly NULLS FIRST;

-- name: DeleteInactiveAnonymousUsers :exec
DELETE FROM users WHERE is_anonymous = true AND last_accessed_at < now() - INTERVAL '30 days';

-- name: GetUserWithPlanInfo :one
SELECT 
    u.id,
    u.firebase_uid,
    u.plan_type,
    u.email,
    u.display_name,
    u.photo_url,
    u.is_anonymous,
    u.last_accessed_at,
    u.created_at,
    u.updated_at,
    pl.max_channels,
    pl.display_name as plan_display_name,
    pl.price_monthly,
    pl.has_favorites,
    pl.has_device_sync,
    pl.description as plan_description
FROM users u
LEFT JOIN plan_limits pl ON u.plan_type = pl.plan_type
WHERE u.id = $1;

