package services

import (
	"context"
	"errors"
	"fmt"
	"main/common"
	"main/db"
	"main/models"
	"main/plugins"
	"strings"
	"text/template"
	"time"
)

type TitleManager struct {
	store         *db.Store
	pluginManager *plugins.PluginManager
	configManager *common.ConfigManager
}

func NewTitleManager(dbStore *db.Store, pluginManager *plugins.PluginManager, configManager *common.ConfigManager) *TitleManager {
	return &TitleManager{
		store:         dbStore,
		pluginManager: pluginManager,
		configManager: configManager,
	}
}

func (m *TitleManager) GetPluginTitle(
	ctx context.Context,
	pluginID string,
	remoteID string,
) (*PluginTitle, error) {
	runtime, err := m.pluginManager.Runtime(pluginID)
	if err != nil {
		return nil, err
	}

	title, err := runtime.GetTitle(remoteID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", plugins.ErrPluginFailedFetch, err)
	}

	result := PluginTitle{
		TitleDetails: *title,
		SavedID:      nil,
	}

	dbTitle, err := m.store.FindSavedTitle(ctx, pluginID, remoteID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return &result, nil
		}
		return nil, err
	}

	result.SavedID = &dbTitle.ID

	return &result, nil
}

func (m *TitleManager) SaveTitle(
	ctx context.Context,
	pluginID string,
	remoteID string,
) (int64, error) {
	plugin, err := m.pluginManager.Runtime(pluginID)
	if err != nil {
		return 0, err
	}

	_, err = m.store.FindSavedTitle(ctx, pluginID, remoteID)
	if err == nil {
		return 0, ErrTitleAlreadySaved
	}
	if !errors.Is(err, db.ErrNotFound) {
		return 0, err
	}

	title, err := plugin.GetTitle(remoteID)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", plugins.ErrPluginFailedFetch, err)
	}

	now := time.Now().UTC()
	savedTitle := db.SavedTitle{
		PluginID:            pluginID,
		RemoteID:            remoteID,
		Title:               title.Title,
		AlternativeTitles:   title.AlternativeTitles,
		Author:              title.Author,
		Artist:              title.Artist,
		Status:              title.Status,
		Description:         title.Description,
		Cover:               title.Cover,
		URL:                 title.URL,
		GroupFilter:         []string{},
		DirectoryName:       title.Title,
		ChapterNameTemplate: "{{if .volume}}v{{.volume}} {{else if .season}}s{{.season}} {{end}}ch.{{.number}}{{if .group}} - {{.group}}{{end}}",
		LastRefreshedAt:     &now,
	}

	var chapters = make([]db.Chapter, 0, len(title.Chapters))
	for _, c := range title.Chapters {
		chapters = append(chapters, c.ConvertToModel(0))
	}

	idCreated, err := m.store.SaveTitle(ctx, savedTitle, chapters)
	if err != nil {
		return 0, err
	}

	return idCreated, nil
}

func (m *TitleManager) RefreshTitle(
	ctx context.Context,
	titleID int64,
) (*RefreshTitleResult, error) {
	title, err := m.store.GetSavedTitle(ctx, titleID)
	if err != nil {
		return nil, err
	}

	plugin, err := m.pluginManager.Runtime(title.PluginID)
	if err != nil {
		return nil, err
	}

	pluginTitle, err := plugin.GetTitle(title.RemoteID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", plugins.ErrPluginFailedFetch, err)
	}

	now := time.Now().UTC()
	title.Title = pluginTitle.Title
	title.AlternativeTitles = pluginTitle.AlternativeTitles
	title.Author = pluginTitle.Author
	title.Artist = pluginTitle.Artist
	title.Status = pluginTitle.Status
	title.Description = pluginTitle.Description
	title.Cover = pluginTitle.Cover
	title.URL = pluginTitle.URL
	title.LastRefreshedAt = &now

	var refreshedChapters = make([]db.Chapter, 0, len(pluginTitle.Chapters))
	for _, c := range pluginTitle.Chapters {
		refreshedChapters = append(refreshedChapters, c.ConvertToModel(title.ID))
	}

	config := m.configManager.Get()

	var oldChapters []db.Chapter
	if config.IgnoreReuploads {
		oldChapters, err = m.store.ListChapters(ctx, title.ID)
		if err != nil {
			return nil, err
		}
	}

	newChapters, err := m.store.RefreshTitle(ctx, title, refreshedChapters)
	if err != nil {
		return nil, err
	}

	chapters, err := m.store.ListChapters(ctx, title.ID)
	if err != nil {
		return nil, err
	}

	outChapters := newChapters
	if config.IgnoreReuploads {
		outChapters = filterNewChapters(oldChapters, newChapters)
	}

	return &RefreshTitleResult{
		Title:       models.ConvertSavedTitle(title, chapters),
		NewChapters: outChapters,
	}, nil
}

func filterNewChapters(old, incoming []db.Chapter) []db.Chapter {
	existing := make(map[float64]struct{}, len(old))

	for _, o := range old {
		existing[o.Number] = struct{}{}
	}

	result := make([]db.Chapter, 0)

	for _, n := range incoming {
		if _, exists := existing[n.Number]; exists {
			continue
		}
		result = append(result, n)
		existing[n.Number] = struct{}{}
	}

	return result
}

func (m *TitleManager) BeginChapterDownload(
	ctx context.Context,
	chapterID int64,
) (*ChapterDownload, error) {
	chapter, err := m.store.GetChapter(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	title, err := m.store.GetSavedTitle(ctx, chapter.SavedTitleID)
	if err != nil {
		return nil, err
	}

	plugin, err := m.pluginManager.Runtime(title.PluginID)
	if err != nil {
		return nil, err
	}

	chapterDesc, err := plugin.GetChapter(chapter.RemoteID)
	if err != nil {
		return nil, err
	}

	nameTmpl, err := template.New("name").Parse(title.ChapterNameTemplate)
	if err != nil {
		return nil, err
	}
	var builder strings.Builder

	templateData := map[string]any{
		"number": chapter.Number,
		"volume": chapter.Volume,
		"season": chapter.Season,
		"group":  chapter.GroupName,
	}
	err = nameTmpl.Execute(&builder, templateData)
	if err != nil {
		return nil, err
	}

	return &ChapterDownload{
		Descriptor:    *chapterDesc,
		Name:          builder.String(),
		DirectoryName: title.DirectoryName,
	}, nil
}

func (m *TitleManager) MarkChapterDownloaded(
	ctx context.Context,
	chapterID int64,
) error {
	chapter, err := m.store.GetChapter(ctx, chapterID)
	if err != nil {
		return err
	}

	chapter.Downloaded = true
	return m.store.UpdateChapter(ctx, chapter)
}
