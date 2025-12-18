-- プラン定義を更新

-- 既存のプランを削除して再作成
DELETE FROM plan_limits;

-- 新しいプラン定義を挿入
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

