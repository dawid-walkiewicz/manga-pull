-- +goose Up

CREATE TABLE plugins_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 0,
    path TEXT NOT NULL
);

INSERT INTO plugins_new (
    id,
    name,
    enabled,
    path
)
SELECT
    id,
    name,
    enabled,
    path
FROM plugins;

DROP TABLE plugins;

ALTER TABLE plugins_new RENAME TO plugins;

-- +goose Down

CREATE TABLE plugins_old (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT '',
    api_version INTEGER NOT NULL DEFAULT 1,
    enabled INTEGER NOT NULL DEFAULT 1,
    path TEXT NOT NULL
);

INSERT INTO plugins_old (
    id,
    name,
    enabled,
    path
)
SELECT
    id,
    name,
    enabled,
    path
FROM plugins;

DROP TABLE plugins;

ALTER TABLE plugins_old RENAME TO plugins;
