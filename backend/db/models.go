package db

import "time"

// TODO : think about what happens when plugin disappears
type Plugin struct {
	ID         string `db:"id"`
	Name       string `db:"name"`
	Version    string `db:"version"`
	APIVersion int    `db:"api_version"`
	Enabled    bool   `db:"enabled"`
	Path       string `db:"path"`
}

type SavedTitle struct {
	ID                  int64      `db:"id"`
	PluginID            string     `db:"plugin_id"`
	RemoteID            string     `db:"remote_id"`
	Title               string     `db:"title"`
	AlternativeTitles   StringList `db:"alternative_titles"`
	Author              *string    `db:"author"`
	Artist              *string    `db:"artist"`
	Status              *string    `db:"status"`
	Description         *string    `db:"description"`
	Cover               *string    `db:"cover"`
	GroupFilter         StringList `db:"group_filter"`
	DirectoryName       string     `db:"directory_name"`
	ChapterNameTemplate string     `db:"chapter_name_template"`
	LastRefreshedAt     *time.Time `db:"last_refreshed_at"`
}

type Chapter struct {
	ID           int64      `db:"id"`
	SavedTitleID int64      `db:"saved_title_id"`
	RemoteID     string     `db:"remote_id"`
	Number       float64    `db:"number"`
	Volume       *int       `db:"volume"`
	Season       *int       `db:"season"`
	Title        *string    `db:"title"`
	GroupName    *string    `db:"group_name"`
	Language     *string    `db:"language"`
	PublishedAt  *time.Time `db:"published_at"`
	Downloaded   bool       `db:"downloaded"`
}

type Job struct {
	ID           int64      `db:"id"`
	JobType      string     `db:"job_type"`
	Status       string     `db:"status"`
	SavedTitleID *int64     `db:"saved_title_id"`
	ChapterID    *int64     `db:"chapter_id"`
	Attempt      int        `db:"attempt"`
	Progress     string     `db:"progress"`
	ErrorMessage *string    `db:"error_message"`
	CreatedAt    time.Time  `db:"created_at"`
	StartedAt    *time.Time `db:"started_at"`
	FinishedAt   *time.Time `db:"finished_at"`
}
