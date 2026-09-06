-- +goose Up
CREATE TABLE job_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    level TEXT NOT NULL DEFAULT 'info',
    message TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE INDEX idx_job_logs_job_id ON job_logs(job_id);

ALTER TABLE jobs
RENAME COLUMN attempt TO retries;

-- +goose Down

DROP TABLE job_logs;

ALTER TABLE jobs
RENAME COLUMN retries TO attempt;
