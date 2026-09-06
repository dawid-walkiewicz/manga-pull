package plugins

import (
	"errors"
	"main/db"
	"time"
)

var (
	ErrPluginNotFound    = errors.New("plugin not found")
	ErrPluginDisabled    = errors.New("plugin disabled")
	ErrPluginRuntime     = errors.New("plugin runtime unavailable")
	ErrPluginFailedFetch = errors.New("failed to fetch title from plugin")
)

type Manifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	APIVersion  int      `json:"apiVersion"`
	Description string   `json:"description"`
	Transport   string   `json:"transport"`
	Domains     []string `json:"domains"`
}

type Plugin struct {
	Manifest

	Path    string  `json:"path"`
	Enabled bool    `json:"enabled"`
	Error   *string `json:"error"`
}

type TitleSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Cover string `json:"cover"`
}

type TitleDetails struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Cover             *string   `json:"cover"`
	AlternativeTitles []string  `json:"alternativeTitles"`
	Author            *string   `json:"author"`
	Artist            *string   `json:"artist"`
	Status            *string   `json:"status"`
	Description       *string   `json:"description"`
	URL               *string   `json:"url"`
	Chapters          []Chapter `json:"chapters"`
}

type Chapter struct {
	ID          string     `json:"id"`
	Number      float64    `json:"number"`
	Volume      *int       `json:"volume"`
	Season      *int       `json:"season"`
	Title       *string    `json:"title"`
	GroupName   *string    `json:"groupName"`
	Language    *string    `json:"language"`
	PublishedAt *time.Time `json:"publishedAt"`
	URL         *string    `json:"url"`
}

func (c *Chapter) ConvertToModel(titleId int64) db.Chapter {
	return db.Chapter{
		SavedTitleID: titleId,
		RemoteID:     c.ID,
		Number:       c.Number,
		Volume:       c.Volume,
		Season:       c.Season,
		Title:        c.Title,
		GroupName:    c.GroupName,
		URL:          c.URL,
		PublishedAt:  c.PublishedAt,
		Downloaded:   false,
	}
}

type PageDescriptor struct {
	Id       string         `json:"id"`
	Url      string         `json:"url"`
	Metadata map[string]any `json:"metadata"`
}

type ChapterDescriptor struct {
	Pages []PageDescriptor `json:"pages"`
}
