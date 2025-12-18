package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// 環境変数ファイルを読み込む
	envFile := ".env.dev"
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("Info: .env file not loaded (%s), using system environment variables", envFile)
	} else {
		log.Printf("✅ Loaded environment from %s", envFile)
	}

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// DB接続
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// 疎通確認
	if err = pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✅ Connected to database successfully!")

	// マイグレーション実行
	log.Println("\n=== Running Migration 001: Create Tables ===")
	if err := runMigration001(ctx, pool); err != nil {
		log.Fatalf("Migration 001 failed: %v", err)
	}
	log.Println("✅ Migration 001 completed")

	log.Println("\n=== Running Migration 002: Seed Platforms ===")
	if err := runMigration002(ctx, pool); err != nil {
		log.Fatalf("Migration 002 failed: %v", err)
	}
	log.Println("✅ Migration 002 completed")

	log.Println("\n=== Running Migration 003: Add Duration to Events ===")
	if err := runMigration003(ctx, pool); err != nil {
		log.Fatalf("Migration 003 failed: %v", err)
	}
	log.Println("✅ Migration 003 completed")

	log.Println("\n=== Running Migration 004: Add is_favorite ===")
	if err := runMigration004(ctx, pool); err != nil {
		log.Fatalf("Migration 004 failed: %v", err)
	}
	log.Println("✅ Migration 004 completed")

	log.Println("\n=== Running Migration 005: Create users and plan_limits ===")
	if err := runMigration005(ctx, pool); err != nil {
		log.Fatalf("Migration 005 failed: %v", err)
	}
	log.Println("✅ Migration 005 completed")

	log.Println("\n=== Running Migration 006: Update Plan Limits ===")
	if err := runMigration006(ctx, pool); err != nil {
		log.Fatalf("Migration 006 failed: %v", err)
	}
	log.Println("✅ Migration 006 completed")

	log.Println("\n🎉 All migrations completed successfully!")
}

func runMigration001(ctx context.Context, pool *pgxpool.Pool) error {
	sql := `
-- ============================================================================
-- platforms: 配信プラットフォーム（YouTube, Twitch等）
-- ============================================================================
CREATE TABLE IF NOT EXISTS platforms (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- sources: チャンネル/配信者
-- ============================================================================
CREATE TABLE IF NOT EXISTS sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_id TEXT NOT NULL REFERENCES platforms(id) ON DELETE RESTRICT,
    external_id TEXT NOT NULL,
    handle TEXT,
    display_name TEXT,
    thumbnail_url TEXT,
    uploads_playlist_id TEXT,
    last_fetched_at TIMESTAMPTZ,
    fetch_status TEXT NOT NULL DEFAULT 'ok',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    UNIQUE (platform_id, external_id)
);

-- ============================================================================
-- user_subscriptions: ユーザーの購読情報
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_subscriptions (
    user_id BIGINT NOT NULL,
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    PRIMARY KEY (user_id, source_id)
);

-- ============================================================================
-- events: タイムライン項目（動画/配信/予定等）
-- ============================================================================
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_id TEXT NOT NULL REFERENCES platforms(id) ON DELETE RESTRICT,
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    external_event_id TEXT NOT NULL,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    start_at TIMESTAMPTZ,
    end_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    url TEXT NOT NULL,
    image_url TEXT,
    metrics JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    UNIQUE (platform_id, external_event_id)
);
`

	_, err := pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// インデックス作成
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_sources_platform_id ON sources(platform_id)",
		"CREATE INDEX IF NOT EXISTS idx_sources_fetch_status ON sources(fetch_status) WHERE fetch_status != 'ok'",
		"CREATE INDEX IF NOT EXISTS idx_user_subscriptions_source_id ON user_subscriptions(source_id)",
		"CREATE INDEX IF NOT EXISTS idx_user_subscriptions_enabled ON user_subscriptions(user_id, enabled) WHERE enabled = true",
		"CREATE INDEX IF NOT EXISTS idx_events_source_published ON events(source_id, published_at DESC NULLS LAST)",
		"CREATE INDEX IF NOT EXISTS idx_events_start_at ON events(start_at DESC NULLS LAST)",
		"CREATE INDEX IF NOT EXISTS idx_events_timeline ON events(source_id, COALESCE(start_at, published_at) DESC NULLS LAST)",
		"CREATE INDEX IF NOT EXISTS idx_events_type ON events(type, start_at DESC NULLS LAST)",
	}

	for _, idx := range indexes {
		if _, err := pool.Exec(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

func runMigration002(ctx context.Context, pool *pgxpool.Pool) error {
	sql := `
INSERT INTO platforms (id, name, created_at) VALUES
    ('youtube', 'YouTube', now()),
    ('twitch', 'Twitch', now()),
    ('podcast', 'Podcast', now()),
    ('niconico', 'ニコニコ生放送', now())
ON CONFLICT (id) DO NOTHING;
`

	_, err := pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to seed platforms: %w", err)
	}

	return nil
}

func runMigration003(ctx context.Context, pool *pgxpool.Pool) error {
	sql := `
ALTER TABLE events ADD COLUMN IF NOT EXISTS duration TEXT;
`

	_, err := pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to add duration column: %w", err)
	}

	return nil
}

func runMigration004(ctx context.Context, pool *pgxpool.Pool) error {
	sql := `
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS is_favorite BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_favorite ON user_subscriptions (user_id, is_favorite) WHERE is_favorite = TRUE;
`

	_, err := pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to add is_favorite: %w", err)
	}

	return nil
}

func runMigration005(ctx context.Context, pool *pgxpool.Pool) error {
	sql := `
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    firebase_uid TEXT UNIQUE NOT NULL,
    plan_type TEXT NOT NULL DEFAULT 'free_anonymous',
    email TEXT,
    display_name TEXT,
    photo_url TEXT,
    is_anonymous BOOLEAN NOT NULL DEFAULT true,
    last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_firebase_uid ON users(firebase_uid);
CREATE INDEX IF NOT EXISTS idx_users_plan_type ON users(plan_type);
CREATE INDEX IF NOT EXISTS idx_users_last_accessed_at ON users(last_accessed_at);

CREATE TABLE IF NOT EXISTS plan_limits (
    plan_type TEXT PRIMARY KEY,
    max_channels INT NOT NULL,
    display_name TEXT NOT NULL,
    price_monthly INT,
    has_favorites BOOLEAN NOT NULL DEFAULT false,
    has_device_sync BOOLEAN NOT NULL DEFAULT false,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO plan_limits (plan_type, max_channels, display_name, price_monthly, has_favorites, has_device_sync, description) VALUES
('free_anonymous', 5, '匿名プラン', NULL, false, false, 'お試しプラン。最終アクセスから30日でデータ削除。広告表示あり。'),
('free_login', 999999, 'ベーシックプラン', NULL, true, true, 'ログインユーザー向け。無制限チャンネル登録、データ永久保存、お気に入り機能、デバイス間同期対応。広告表示あり。'),
('pro', 999999, 'プロプラン', 500, true, true, '広告なし。無制限チャンネル登録、全機能利用可能。')
ON CONFLICT (plan_type) DO UPDATE SET max_channels = EXCLUDED.max_channels, display_name = EXCLUDED.display_name, description = EXCLUDED.description;

ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_last_accessed ON user_subscriptions(last_accessed_at);
`

	_, err := pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to create users and plan_limits: %w", err)
	}

	return nil
}

func runMigration006(ctx context.Context, pool *pgxpool.Pool) error {
	sql := `
-- プラン定義を更新
DELETE FROM plan_limits;

INSERT INTO plan_limits (plan_type, max_channels, display_name, price_monthly, has_favorites, has_device_sync, description) VALUES
('free_anonymous', 5, 'Free（匿名）', NULL, false, false, 'ログイン不要・まずはお試し。最大5チャンネル登録、データ保持30日。'),
('free_login', 20, 'Basic（ログイン）', NULL, true, true, '標準プラン。最大20チャンネル登録、データ無制限保持、マルチデバイス同期、お気に入り機能。'),
('plus', 999999, 'Plus（課金）', 500, true, true, 'ヘビーユーザー向け。無制限チャンネル登録、全機能利用可能。')
ON CONFLICT (plan_type) DO UPDATE SET 
    max_channels = EXCLUDED.max_channels,
    display_name = EXCLUDED.display_name,
    price_monthly = EXCLUDED.price_monthly,
    has_favorites = EXCLUDED.has_favorites,
    has_device_sync = EXCLUDED.has_device_sync,
    description = EXCLUDED.description;
`

	_, err := pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to update plan limits: %w", err)
	}

	return nil
}
