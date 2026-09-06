package services

import (
	"main/db"
	"main/models"
	"main/plugins"
)

type PluginTitle struct {
	plugins.TitleDetails

	SavedID *int64 `json:"savedId"`
}

type RefreshTitleResult struct {
	Title       models.SavedTitle
	NewChapters []db.Chapter
}

type ChapterDownload struct {
	Descriptor    plugins.ChapterDescriptor
	Name          string
	DirectoryName string
}
