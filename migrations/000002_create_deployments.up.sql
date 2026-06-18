CREATE TABLE IF NOT EXISTS deployments (
    id          SERIAL PRIMARY KEY,
    service     TEXT NOT NULL,
    image       TEXT NOT NULL,
    deployed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deployments_service ON deployments(service);
CREATE INDEX idx_deployments_deployed_at ON deployments(deployed_at DESC);
