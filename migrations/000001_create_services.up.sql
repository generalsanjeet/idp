CREATE TABLE IF NOT EXISTS services (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    repo_url   TEXT NOT NULL,
    owner      TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
