-- Per-user run/log retention. NULL = keep forever (the default, matching prior
-- behaviour). When set, a background job deletes runs — and the logs embedded in
-- their encrypted payload — older than this many hours.
ALTER TABLE users ADD COLUMN retention_hours INT;

-- Speeds up the age-based prune sweep.
CREATE INDEX IF NOT EXISTS idx_runs_received_at ON runs (received_at);
