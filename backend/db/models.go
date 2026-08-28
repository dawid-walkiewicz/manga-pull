package db

import "time"

type PluginRecord struct {
	ID      string `db:"id"`
	Name    string `db:"name"`
	Enabled bool   `db:"enabled"`
	Path    string `db:"path"`
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
	URL                 *string    `db:"url"`
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
	URL          *string    `db:"url"`
	PublishedAt  *time.Time `db:"published_at"`
	Downloaded   bool       `db:"downloaded"`
}

type Job struct {
	ID           int64      `db:"id" json:"id"`
	JobType      string     `db:"job_type" json:"jobType"`
	Status       string     `db:"status" json:"status"`
	SavedTitleID *int64     `db:"saved_title_id" json:"savedTitleId"`
	ChapterID    *int64     `db:"chapter_id" json:"chapterId"`
	Attempt      int        `db:"attempt" json:"attempt"`
	Progress     string     `db:"progress" json:"progress"`
	ErrorMessage *string    `db:"error_message" json:"errorMessage"`
	CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
	StartedAt    *time.Time `db:"started_at" json:"startedAt"`
	FinishedAt   *time.Time `db:"finished_at" json:"finishedAt"`
}
