package services

import (
	"context"
	"fmt"
	"main/db"
	"main/models"
	"main/plugins"
	"math"
	"time"
)

type PluginManager struct {
	store *db.Store
	dir   string

	plugins  []plugins.Plugin
	runtimes *plugins.RuntimeRegistry
}

func NewPluginManager(store *db.Store, dir string) *PluginManager {
	return &PluginManager{
		store:    store,
		dir:      dir,
		plugins:  make([]plugins.Plugin, 0),
		runtimes: plugins.NewRuntimeRegistry(),
	}
}

func convertPlugins(plugins []plugins.Plugin) []db.PluginRecord {
	conv_plugins := make([]db.PluginRecord, 0)
	for _, p := range plugins {
		conv_plugins = append(conv_plugins, db.PluginRecord{
			ID:      p.ID,
			Name:    p.Name,
			Enabled: false,
			Path:    p.Path,
		})
	}
	return conv_plugins
}

func (m *PluginManager) findPlugin(id string) *plugins.Plugin {
	for i := range m.plugins {
		if m.plugins[i].ID == id {
			return &m.plugins[i]
		}
	}

	return nil
}

func (m *PluginManager) Plugin(id string) (*plugins.Plugin, bool) {
	plugin := m.findPlugin(id)
	return plugin, plugin != nil
}

func applyPluginState(
	discovered []plugins.Plugin,
	records []db.PluginRecord,
) []plugins.Plugin {
	byID := make(map[string]db.PluginRecord, len(records))

	for _, record := range records {
		byID[record.ID] = record
	}

	for i := range discovered {
		if record, ok := byID[discovered[i].ID]; ok {
			discovered[i].Enabled = record.Enabled
		}
	}

	return discovered
}

func (m *PluginManager) Scan(ctx context.Context) error {
	paths, err := plugins.FindPluginZips(m.dir)
	if err != nil {
		return fmt.Errorf("find plugins: %w", err)
	}

	discovered := plugins.LoadPlugins(paths)
	items := convertPlugins(discovered)

	if err := m.store.CreatePlugins(ctx, items); err != nil {
		return fmt.Errorf("create plugins: %w", err)
	}

	records, err := m.store.ListPlugins(ctx)
	if err != nil {
		return fmt.Errorf("list plugins: %w", err)
	}

	m.plugins = applyPluginState(discovered, records)

	return nil
}

func (m *PluginManager) Start(ctx context.Context) error {
	if err := m.Scan(ctx); err != nil {
		return err
	}

	for i := range m.plugins {
		plugin := &m.plugins[i]

		if !plugin.Enabled {
			continue
		}

		runtime, err := plugins.NewRuntime(plugin)
		if err != nil {
			return fmt.Errorf(
				"start plugin %q: %w",
				plugin.ID,
				err,
			)
		}

		m.runtimes.Register(runtime)
	}

	return nil
}

func (m *PluginManager) Enable(ctx context.Context, id string) error {
	plugin := m.findPlugin(id)
	if plugin == nil {
		return fmt.Errorf("%w: %s", ErrPluginNotFound, id)
	}

	if plugin.Enabled {
		return nil
	}

	runtime, err := plugins.NewRuntime(plugin)
	if err != nil {
		return err
	}

	m.runtimes.Register(runtime)

	if err := m.store.SetPluginEnabled(ctx, id, true); err != nil {
		m.runtimes.Unregister(id)
		return err
	}

	plugin.Enabled = true
	return nil
}

func (m *PluginManager) Disable(ctx context.Context, id string) error {
	plugin := m.findPlugin(id)
	if plugin == nil {
		return fmt.Errorf("%w: %s", ErrPluginNotFound, id)
	}

	if !plugin.Enabled {
		return nil
	}

	if err := m.store.SetPluginEnabled(ctx, id, false); err != nil {
		return err
	}

	m.runtimes.Unregister(id)

	plugin.Enabled = false
	return nil
}

func (m *PluginManager) Plugins() []plugins.Plugin {
	return m.plugins
}

func (m *PluginManager) Runtime(id string) (*plugins.PluginRuntime, error) {
	runtime, ok := m.runtimes.Get(id)
	if ok {
		return runtime, nil
	}

	plugin, ok := m.Plugin(id)
	if !ok {
		return nil, ErrPluginNotFound
	}

	if !plugin.Enabled {
		return nil, ErrPluginDisabled
	}

	return nil, ErrPluginRuntime
}

func (m *PluginManager) SaveTitle(
	ctx context.Context,
	pluginID string,
	remoteID string,
) (int64, error) {
	plugin, err := m.Runtime(pluginID)
	if err != nil {
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

func (m *PluginManager) RefreshTitle(
	ctx context.Context,
	pluginID string,
	remoteID string,
) (*models.SavedTitle, error) {
	plugin, err := m.Runtime(pluginID)
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
