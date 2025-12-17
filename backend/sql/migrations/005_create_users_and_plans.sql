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

