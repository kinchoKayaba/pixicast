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
