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