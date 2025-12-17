-- Migration: 001_create_tables
-- Description: Create core tables for Pixicast (platforms, sources, user_subscriptions, events)
-- Compatible with: PostgreSQL 12+ / CockroachDB 21+

-- ============================================================================
-- platforms: 配信プラットフォーム（YouTube, Twitch等）
-- ============================================================================
CREATE TABLE platforms (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- sources: チャンネル/配信者
-- ============================================================================
CREATE TABLE sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_id TEXT NOT NULL REFERENCES platforms(id) ON DELETE RESTRICT,
    external_id TEXT NOT NULL,  -- 例: YouTube channelId (UCxxx...)
    handle TEXT,                -- 例: @junchannel (nullable)
    display_name TEXT,
    thumbnail_url TEXT,
    uploads_playlist_id TEXT,   -- YouTube用: UUxxx... (nullable)
    last_fetched_at TIMESTAMPTZ,
    fetch_status TEXT NOT NULL DEFAULT 'ok',  -- ok / not_found / suspended / error
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    UNIQUE (platform_id, external_id)
);

-- インデックス: プラットフォーム別検索用
CREATE INDEX idx_sources_platform_id ON sources(platform_id);

-- インデックス: 取り込みステータス確認用
CREATE INDEX idx_sources_fetch_status ON sources(fetch_status) WHERE fetch_status != 'ok';

-- ============================================================================
-- user_subscriptions: ユーザーの購読情報
-- ============================================================================
CREATE TABLE user_subscriptions (
    user_id BIGINT NOT NULL,    -- 将来usersテーブルとFK設定予定
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority INT NOT NULL DEFAULT 0,  -- 表示順序用（大きいほど優先）
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    PRIMARY KEY (user_id, source_id)
);

-- インデックス: source_id から購読者を検索
CREATE INDEX idx_user_subscriptions_source_id ON user_subscriptions(source_id);

-- インデックス: 有効な購読のみフィルタ
CREATE INDEX idx_user_subscriptions_enabled ON user_subscriptions(user_id, enabled) WHERE enabled = true;

-- ============================================================================
-- events: タイムライン項目（動画/配信/予定等）
-- ============================================================================
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_id TEXT NOT NULL REFERENCES platforms(id) ON DELETE RESTRICT,
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    external_event_id TEXT NOT NULL,  -- 例: YouTube videoId
    type TEXT NOT NULL,               -- live / scheduled / video / premiere
    title TEXT NOT NULL,
    description TEXT,
    start_at TIMESTAMPTZ,             -- 配信開始時刻（ライブ/予定のみ）
    end_at TIMESTAMPTZ,               -- 配信終了時刻（ライブのみ）
    published_at TIMESTAMPTZ,         -- 公開日時（動画等）
    url TEXT NOT NULL,
    image_url TEXT,
    metrics JSONB,                    -- {"views": 123, "likes": 45, ...}
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    UNIQUE (platform_id, external_event_id)
);

-- インデックス: source別タイムライン取得用
CREATE INDEX idx_events_source_published ON events(source_id, published_at DESC NULLS LAST);

-- インデックス: 開始時刻順でのソート用（ライブ/予定）
CREATE INDEX idx_events_start_at ON events(start_at DESC NULLS LAST);

-- インデックス: タイムライン全体の取得用（複合条件）
CREATE INDEX idx_events_timeline ON events(
    source_id,
    COALESCE(start_at, published_at) DESC NULLS LAST
);

-- インデックス: イベントタイプ別検索用
CREATE INDEX idx_events_type ON events(type, start_at DESC NULLS LAST);

-- ============================================================================
-- コメント
-- ============================================================================
COMMENT ON TABLE platforms IS '配信プラットフォーム（YouTube, Twitch等）';
COMMENT ON TABLE sources IS 'チャンネル/配信者の情報';
COMMENT ON TABLE user_subscriptions IS 'ユーザーの購読情報';
COMMENT ON TABLE events IS 'タイムライン項目（動画/配信/予定等）';

COMMENT ON COLUMN sources.fetch_status IS 'ok=正常, not_found=削除/非公開, suspended=BAN, error=取得エラー';
COMMENT ON COLUMN events.type IS 'live=配信中, scheduled=予定, video=アーカイブ動画, premiere=プレミア公開';
COMMENT ON COLUMN events.metrics IS 'JSON形式の統計情報 {"views": 123, "likes": 45, "comments": 67}';

