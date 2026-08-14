-- +goose Up

CREATE TABLE plugins (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    api_version INTEGER NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    path TEXT NOT NULL
);

CREATE TABLE saved_titles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plugin_id TEXT NOT NULL,
    remote_id TEXT NOT NULL,
    title TEXT NOT NULL,
    alternative_titles TEXT NOT NULL DEFAULT '[]',
    author TEXT,
    artist TEXT,
    status TEXT,
    description TEXT,
    cover TEXT,
    group_filter TEXT NOT NULL DEFAULT '[]',
    directory_name TEXT NOT NULL,
    chapter_name_template TEXT NOT NULL,
    last_refreshed_at DATETIME,

    FOREIGN KEY (plugin_id)
        REFERENCES plugins(id)
        ON DELETE CASCADE,

    UNIQUE(plugin_id, remote_id)
);

CREATE TABLE chapters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    saved_title_id INTEGER NOT NULL,
    remote_id TEXT NOT NULL,
    number REAL NOT NULL,
    volume INTEGER,
    season INTEGER,
    title TEXT,
    group_name TEXT,
    language TEXT,
    published_at DATETIME,
    downloaded INTEGER NOT NULL DEFAULT 0,

    FOREIGN KEY (saved_title_id)
        REFERENCES saved_titles(id)
        ON DELETE CASCADE
);

CREATE TABLE jobs (
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

CREATE INDEX idx_saved_titles_plugin_id
    ON saved_titles(plugin_id);

CREATE INDEX idx_chapters_saved_title_id
    ON chapters(saved_title_id);

CREATE INDEX idx_jobs_status
    ON jobs(status);

CREATE INDEX idx_jobs_saved_title_id
    ON jobs(saved_title_id);


-- +goose Down

DROP TABLE jobs;
DROP TABLE chapters;
DROP TABLE saved_titles;
DROP TABLE plugins;
