-- Migration: 002_seed_platforms
-- Description: Insert initial platform data
-- Compatible with: PostgreSQL 12+ / CockroachDB 21+

-- ============================================================================
-- 初期プラットフォームデータ
-- ============================================================================

INSERT INTO platforms (id, name, created_at) VALUES
    ('youtube', 'YouTube', now()),
    ('twitch', 'Twitch', now()),
    ('niconico', 'ニコニコ生放送', now())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- コメント
-- ============================================================================
-- youtube: YouTube Data API v3 を使用
-- twitch: Twitch API (Helix) を使用
-- niconico: ニコニコ生放送API を使用（将来実装予定）

