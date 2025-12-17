-- 003_add_duration_to_events.sql
-- Add duration field to events table for storing video/stream duration

ALTER TABLE events ADD COLUMN IF NOT EXISTS duration TEXT;

COMMENT ON COLUMN events.duration IS 'Video/stream duration in HH:MM:SS or MM:SS format';

