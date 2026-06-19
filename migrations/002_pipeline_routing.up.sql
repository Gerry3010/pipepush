-- Routing key lets a project-scoped token resolve a pipeline by the (plaintext)
-- pipeline name sent in a webhook, without the server ever decrypting names.
-- It is a deterministic hash of the normalized name, computed identically by
-- the client (on `pipelines create`) and the server (on project-token webhooks).
ALTER TABLE pipelines ADD COLUMN routing_key CHAR(64);

-- One pipeline per (project, routing_key) so project-token routing is unambiguous
-- and auto-create stays idempotent. NULL keys (legacy/pipeline-token-only) are exempt.
CREATE UNIQUE INDEX idx_pipelines_project_routing_key
    ON pipelines (project_id, routing_key)
    WHERE routing_key IS NOT NULL;
