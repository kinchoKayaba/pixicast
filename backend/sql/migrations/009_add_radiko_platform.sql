-- Migration: 009_add_radiko_platform
-- Description: Add Radiko platform support
-- Compatible with: PostgreSQL 12+ / CockroachDB 21+

-- ============================================================================
-- Add Radiko platform
-- ============================================================================
INSERT INTO platforms (id, name, created_at)
VALUES ('radiko', 'Radiko', now())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- Add radio event type support
-- ============================================================================
COMMENT ON COLUMN events.type IS 'live=配信中, scheduled=予定, video=アーカイブ動画, premiere=プレミア公開, radio=ラジオ番組';
