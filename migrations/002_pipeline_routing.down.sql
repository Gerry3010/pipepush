DROP INDEX IF EXISTS idx_pipelines_project_routing_key;
ALTER TABLE pipelines DROP COLUMN IF EXISTS routing_key;
