-- +goose Up
CREATE TABLE jobs(
id TEXT PRIMARY KEY,
once TIMESTAMP NULL,
interval TEXT NULL,
status TEXT NOT NULL,
created_at BIGINT NOT NULL,
last_finished_at BIGINT,
payload JSONB
);

CREATE TABLE executions (
id TEXT PRIMARY KEY,
job_id TEXT REFERENCES jobs(id),
worker_id TEXT,
status TEXT,
started_at BIGINT,
finished_at BIGINT
);

-- +goose Down
DROP TABLE executions;
DROP TABLE jobs;