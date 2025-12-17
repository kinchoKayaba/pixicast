CREATE TABLE programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title STRING NOT NULL,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    platform_name STRING NOT NULL,
    image_url STRING,
    link_url STRING,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- 配信元（YouTubeチャンネル等）
CREATE TABLE sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_id STRING NOT NULL, -- 'youtube' etc.
    external_id STRING NOT NULL, -- YouTubeのchannelId (UCxxx...)
    handle STRING,               -- @junchannel etc. (オプション)
    display_name STRING,
    thumbnail_url STRING,
    uploads_playlist_id STRING,  -- YouTubeのUUxxx... プレイリストID
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (platform_id, external_id)
);

-- ユーザー購読
CREATE TABLE user_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INT NOT NULL,        -- 認証実装後にusersテーブルと外部キー設定
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    enabled BOOL NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (user_id, source_id)
);