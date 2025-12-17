ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS is_favorite BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_favorite ON user_subscriptions (user_id, is_favorite) WHERE is_favorite = TRUE;

