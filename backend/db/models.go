package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

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
	ID           int64       `db:"id" json:"id"`
	JobType      string      `db:"job_type" json:"jobType"`
	Status       string      `db:"status" json:"status"`
	Payload      JSONPayload `db:"payload" json:"payload"`
	Retries      int         `db:"retries" json:"retries"`
	Progress     string      `db:"progress" json:"progress"`
	ErrorMessage *string     `db:"error_message" json:"errorMessage"`
	CreatedAt    time.Time   `db:"created_at" json:"createdAt"`
	StartedAt    *time.Time  `db:"started_at" json:"startedAt"`
	FinishedAt   *time.Time  `db:"finished_at" json:"finishedAt"`
}

type JSONPayload json.RawMessage

func (p *JSONPayload) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*p = JSONPayload(`{}`)
		return nil
	case string:
		*p = JSONPayload(v)
		return nil
	case []byte:
		*p = JSONPayload(v)
		return nil
	default:
		return fmt.Errorf("unsupported JSONPayload type %T", value)
	}
}

func (p JSONPayload) Value() (driver.Value, error) {
	if len(p) == 0 {
		return "{}", nil
	}

	if !json.Valid(p) {
		return nil, fmt.Errorf("invalid JSON payload")
	}

	return string(p), nil
}

type JobLog struct {
	ID        int64     `db:"id" json:"id"`
	JobID     int64     `db:"job_id" json:"jobId"`
	Level     string    `db:"level" json:"level"`
	Message   string    `db:"message" json:"message"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}
