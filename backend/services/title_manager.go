package services

import (
	"context"
	"errors"
	"fmt"
	"main/db"
	"main/models"
	"math"
	"time"
)

type TitleManager struct {
	store         *db.Store
	pluginManager *PluginManager
}

func NewTitleManager(dbStore *db.Store, pluginManager *PluginManager) *TitleManager {
	return &TitleManager{
		store:         dbStore,
		pluginManager: pluginManager,
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
		return nil, fmt.Errorf("%w: %v", ErrPluginFailedFetch, err)
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
		return 0, fmt.Errorf("%w: %v", ErrPluginFailedFetch, err)
	}

	now := time.Now()
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
	pluginID string,
	remoteID string,
) (*models.SavedTitle, error) {
	plugin, err := m.pluginManager.Runtime(pluginID)
	if err != nil {
		return nil, err
	}

	dbTitle, err := m.store.FindSavedTitle(ctx, pluginID, remoteID)
	if err != nil {
		return nil, err
	}

	title, err := plugin.GetTitle(remoteID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPluginFailedFetch, err)
	}

	now := time.Now()
	dbTitle.Title = title.Title
	dbTitle.AlternativeTitles = title.AlternativeTitles
	dbTitle.Author = title.Author
	dbTitle.Artist = title.Artist
	dbTitle.Status = title.Status
	dbTitle.Description = title.Description
	dbTitle.Cover = title.Cover
	dbTitle.URL = title.URL
	dbTitle.LastRefreshedAt = &now

	var refreshedChapters = make([]db.Chapter, 0, len(title.Chapters))
	for _, c := range title.Chapters {
		refreshedChapters = append(refreshedChapters, c.ConvertToModel(dbTitle.ID))
	}

	// oldChapters, err := m.store.ListChapters(ctx, dbTitle.ID)
	// if err != nil {
	// 	return nil, err
	// }

	_, err = m.store.RefreshTitle(ctx, dbTitle, refreshedChapters)
	if err != nil {
		return nil, err
	}

	chapters, err := m.store.ListChapters(ctx, dbTitle.ID)
	if err != nil {
		return nil, err
	}

	refreshedTitle := models.ConvertSavedTitle(dbTitle, chapters)
	return &refreshedTitle, nil
}

func filterNewChapters(old, incoming []db.Chapter) []db.Chapter {
	var max float64
	for _, ch := range old {
		max = math.Max(ch.Number, max)
	}

	result := make([]db.Chapter, 0)

	for _, ch := range incoming {
		if ch.Number > max {
			result = append(result, ch)
		}
	}

	return result
}
