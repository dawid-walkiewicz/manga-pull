package models

import (
	"time"

	"main/db"
)

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
	URL                 *string    `json:"url"`
	GroupFilter         []string   `json:"groupFilter"`
	DirectoryName       string     `json:"directoryName"`
	ChapterNameTemplate string     `json:"chapterNameTemplate"`
	LastRefreshedAt     *time.Time `json:"lastRefreshedAt"`

	Chapters []Chapter `json:"chapters"`
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
	URL          *string    `json:"url"`
	PublishedAt  *time.Time `json:"publishedAt"`
	Downloaded   bool       `json:"downloaded"`
}

func ConvertSavedTitle(title db.SavedTitle, chapters []db.Chapter) SavedTitle {
	responseChapters := make([]Chapter, len(chapters))
	for i, c := range chapters {
		responseChapters[i] = Chapter{
			ID:           c.ID,
			SavedTitleID: c.SavedTitleID,
			RemoteID:     c.RemoteID,
			Number:       c.Number,
			Volume:       c.Volume,
			Season:       c.Season,
			Title:        c.Title,
			GroupName:    c.GroupName,
			URL:          c.URL,
			PublishedAt:  c.PublishedAt,
			Downloaded:   c.Downloaded,
		}
	}

	titleWithChapters := SavedTitle{
		ID:                  title.ID,
		PluginID:            title.PluginID,
		RemoteID:            title.RemoteID,
		Title:               title.Title,
		AlternativeTitles:   title.AlternativeTitles,
		Author:              title.Author,
		Artist:              title.Artist,
		Status:              title.Status,
		Description:         title.Description,
		Cover:               title.Cover,
		URL:                 title.URL,
		GroupFilter:         title.GroupFilter,
		DirectoryName:       title.DirectoryName,
		ChapterNameTemplate: title.ChapterNameTemplate,
		LastRefreshedAt:     title.LastRefreshedAt,
		Chapters:            responseChapters,
	}

	return titleWithChapters
}

type SavedTitleSummary struct {
	ID       int64   `json:"id"`
	PluginID string  `json:"pluginId"`
	RemoteID string  `json:"remoteId"`
	Title    string  `json:"title"`
	Cover    *string `json:"cover"`
}

func ConvertSavedTitleToSummary(title db.SavedTitle) SavedTitleSummary {
	summary := SavedTitleSummary{
		ID:       title.ID,
		PluginID: title.PluginID,
		RemoteID: title.RemoteID,
		Title:    title.Title,
		Cover:    title.Cover,
	}

	return summary
}
