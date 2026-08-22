package models

import "time"

type SavedTitle struct {
	ID                  int64      `json:"id"`
	PluginID            string     `json:"pluginId"`
	RemoteID            string     `json:"remoteId"`
	Title               string     `json:"title"`
	AlternativeTitles   []string   `json:"alternativeTitles"`
	Author              *string    `json:"author"`
	Artist              *string    `json:"artist"`
	Status              *string    `json:"status"`
	Description         *string    `json:"description"`
	Cover               *string    `json:"cover"`
	GroupFilter         []string   `json:"groupFilter"`
	DirectoryName       string     `json:"directoryName"`
	ChapterNameTemplate string     `json:"chapterNameTemplate"`
	LastRefreshedAt     *time.Time `json:"lastRefreshedAt"`

	Chapters []Chapter
}

type Chapter struct {
	ID           int64      `json:"id"`
	SavedTitleID int64      `json:"savedTitleId"`
	RemoteID     string     `json:"remoteId"`
	Number       float64    `json:"number"`
	Volume       *int       `json:"volume"`
	Season       *int       `json:"season"`
	Title        *string    `json:"title"`
	GroupName    *string    `json:"groupName"`
	Language     *string    `json:"language"`
	PublishedAt  *time.Time `json:"publishedAt"`
	Downloaded   bool       `json:"downloaded"`
}
