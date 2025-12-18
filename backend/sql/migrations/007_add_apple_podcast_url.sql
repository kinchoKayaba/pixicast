-- Add apple_podcast_url column to sources table
ALTER TABLE sources ADD COLUMN IF NOT EXISTS apple_podcast_url TEXT;

-- Add comment
COMMENT ON COLUMN sources.apple_podcast_url IS 'Apple Podcasts URL (for podcast platform only)';

