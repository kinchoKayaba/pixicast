-- Migration: 010_add_podcast_platform
-- Description: Add Podcast platform to platforms table
-- Compatible with: PostgreSQL 12+ / CockroachDB 21+

INSERT INTO platforms (id, name, created_at)
VALUES ('podcast', 'Podcast', now())
ON CONFLICT (id) DO NOTHING;
