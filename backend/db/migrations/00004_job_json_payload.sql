-- +goose Up

CREATE TABLE jobs_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_type TEXT NOT NULL,
    status TEXT NOT NULL,
    payload TEXT NOT NULL DEFAULT '{}',
    attempt INTEGER NOT NULL DEFAULT 0,
    progress TEXT NOT NULL DEFAULT '',
    error_message TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    finished_at DATETIME
);

INSERT INTO jobs_new (
    id,
    job_type,
    status,
    payload,
    attempt,
    progress,
    error_message,
    created_at,
    started_at,
    finished_at
)
SELECT
    id,
    job_type,
    status,
    CASE
        WHEN chapter_id IS NOT NULL THEN
            json_object('chapterId', chapter_id)
        WHEN saved_title_id IS NOT NULL THEN
            json_object('savedTitleId', saved_title_id)
        ELSE
            '{}'
    END,
    attempt,
    progress,
    error_message,
    created_at,
    started_at,
    finished_at
FROM jobs;

DROP TABLE jobs;

ALTER TABLE jobs_new RENAME TO jobs;

CREATE INDEX idx_jobs_status
ON jobs(status);


-- +goose Down

CREATE TABLE jobs_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_type TEXT NOT NULL,
    status TEXT NOT NULL,
    saved_title_id INTEGER,
    chapter_id INTEGER,
    attempt INTEGER NOT NULL DEFAULT 0,
    progress TEXT NOT NULL DEFAULT '',
    error_message TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    finished_at DATETIME,
    FOREIGN KEY (saved_title_id)
        REFERENCES saved_titles(id)
        ON DELETE SET NULL,
    FOREIGN KEY (chapter_id)
        REFERENCES chapters(id)
        ON DELETE SET NULL
);

INSERT INTO jobs_old (
    id,
    job_type,
    status,
    saved_title_id,
    chapter_id,
    attempt,
    progress,
    error_message,
    created_at,
    started_at,
    finished_at
)
SELECT
    id,
    job_type,
    status,
    json_extract(payload, '$.savedTitleId'),
    json_extract(payload, '$.chapterId'),
    attempt,
    progress,
    error_message,
    created_at,
    started_at,
    finished_at
FROM jobs;

DROP TABLE jobs;

ALTER TABLE jobs_old RENAME TO jobs;

CREATE INDEX idx_jobs_status
ON jobs(status);

CREATE INDEX idx_jobs_saved_title_id
ON jobs(saved_title_id);
