-- Migration: 006_add_source_priority
-- Description: Add source_priority table and update_schedule for batch optimization
-- Compatible with: PostgreSQL 12+ / CockroachDB 21+

-- ============================================================================
-- source_priority: チャンネル優先度管理テーブル
-- ============================================================================
CREATE TABLE IF NOT EXISTS source_priority (
    source_id UUID PRIMARY KEY REFERENCES sources(id) ON DELETE CASCADE,
    subscriber_count INT NOT NULL DEFAULT 0,
    popularity_ratio DECIMAL(5, 4) NOT NULL DEFAULT 0.0000,
    priority_level TEXT NOT NULL DEFAULT 'low',  -- high / medium / low
    update_interval_minutes INT NOT NULL DEFAULT 360,  -- 更新間隔（分）
    last_calculated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- インデックス: 優先度レベル別検索用
CREATE INDEX IF NOT EXISTS idx_source_priority_level ON source_priority(priority_level);

-- インデックス: 最終計算日時順
CREATE INDEX IF NOT EXISTS idx_source_priority_calculated ON source_priority(last_calculated_at);

-- ============================================================================
-- update_schedule: 更新スケジュール管理テーブル
-- ============================================================================
CREATE TABLE IF NOT EXISTS update_schedule (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMPTZ NOT NULL,
    priority_level TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',  -- pending / running / completed / failed
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- インデックス: スケジュール時刻順
CREATE INDEX IF NOT EXISTS idx_update_schedule_scheduled ON update_schedule(scheduled_at);

-- インデックス: ステータス別検索用
CREATE INDEX IF NOT EXISTS idx_update_schedule_status ON update_schedule(status) WHERE status != 'completed';

-- インデックス: source_id別検索用
CREATE INDEX IF NOT EXISTS idx_update_schedule_source ON update_schedule(source_id);

-- ============================================================================
-- api_quota_usage: API使用量追跡テーブル
-- ============================================================================
CREATE TABLE IF NOT EXISTS api_quota_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL,
    platform_id TEXT NOT NULL REFERENCES platforms(id) ON DELETE RESTRICT,
    endpoint TEXT NOT NULL,
    quota_cost INT NOT NULL,
    request_count INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- インデックス: 日付別集計用
CREATE INDEX IF NOT EXISTS idx_api_quota_date ON api_quota_usage(date, platform_id);

-- インデックス: プラットフォーム別集計用
CREATE INDEX IF NOT EXISTS idx_api_quota_platform ON api_quota_usage(platform_id, date);

-- ユニーク制約: 同じ日・プラットフォーム・エンドポイントの重複防止
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_quota_unique ON api_quota_usage(date, platform_id, endpoint);

-- ============================================================================
-- コメント
-- ============================================================================
COMMENT ON TABLE source_priority IS 'チャンネル優先度管理（バッチ処理最適化用）';
COMMENT ON TABLE update_schedule IS '更新スケジュール管理';
COMMENT ON TABLE api_quota_usage IS 'API使用量追跡（YouTube等）';

COMMENT ON COLUMN source_priority.priority_level IS 'high=1h更新, medium=3h更新, low=6h更新';
COMMENT ON COLUMN source_priority.update_interval_minutes IS '更新間隔（分）: high=60, medium=180, low=360';
COMMENT ON COLUMN update_schedule.status IS 'pending=待機中, running=実行中, completed=完了, failed=失敗';
