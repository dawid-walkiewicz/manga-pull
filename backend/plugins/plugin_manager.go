package plugins

import (
	"context"
	"fmt"
	"log"

	"main/db"
)

type PluginManager struct {
	store *db.Store
	dir   string

	plugins  map[string]*Plugin
	runtimes *RuntimeRegistry
}

func NewPluginManager(store *db.Store, dir string) *PluginManager {
	return &PluginManager{
		store:    store,
		dir:      dir,
		plugins:  make(map[string]*Plugin),
		runtimes: nil,
	}
}

func convertPlugins(plugins []Plugin) []db.PluginRecord {
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

func (m *PluginManager) Plugin(id string) (*Plugin, bool) {
	plugin, ok := m.plugins[id]
	return plugin, ok
}

func (m *PluginManager) Scan(ctx context.Context) error {
	paths, err := FindPluginZips(m.dir)
	if err != nil {
		return fmt.Errorf("find plugins: %w", err)
	}

	discovered := LoadPlugins(paths)
	items := convertPlugins(discovered)

	if err := m.store.InsertPlugins(ctx, items); err != nil {
		return fmt.Errorf("insert plugins: %w", err)
	}

	records, err := m.store.ListPlugins(ctx)
	if err != nil {
		return fmt.Errorf("list plugins: %w", err)
	}

	recordsByID := make(map[string]db.PluginRecord, len(records))

	for _, record := range records {
		recordsByID[record.ID] = record
	}

	registry := NewRuntimeRegistry()

	for _, dp := range discovered {
		if r, ok := recordsByID[dp.ID]; ok {
			dp.Enabled = r.Enabled
		}
		m.plugins[dp.ID] = &dp
		plugin := m.plugins[dp.ID]
		delete(recordsByID, dp.ID)

		if plugin.Enabled {
			runtime, err := NewRuntime(plugin)
			if err != nil {
				errMsg := err.Error()
				plugin.Error = &errMsg
				log.Println(fmt.Errorf(
					"start plugin %q: %w",
					plugin.ID,
					err,
				))
				continue
			}

			plugin.Error = nil
			registry.Register(plugin.ID, runtime)
		}
	}

	for _, r := range recordsByID {
		errStr := fmt.Sprintf("plugin not found: %s", r.ID)
		m.plugins[r.ID] = &Plugin{
			Enabled: false,
			Error:   &errStr,
		}
	}

	m.runtimes = registry

	return nil
}

func (m *PluginManager) Enable(ctx context.Context, id string) error {
	plugin, ok := m.plugins[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrPluginNotFound, id)
	}

	if plugin.Enabled {
		return nil
	}

	if plugin.Error != nil {
		return fmt.Errorf("%w: %s", ErrPluginRuntime, *plugin.Error)
	}

	runtime, err := NewRuntime(plugin)
	if err != nil {
		errMsg := err.Error()
		plugin.Error = &errMsg
		return fmt.Errorf("%w: %v", ErrPluginRuntime, err)
	}

	m.runtimes.Register(plugin.ID, runtime)

	if err := m.store.SetPluginEnabled(ctx, id, true); err != nil {
		m.runtimes.Unregister(id)
		return err
	}

	plugin.Enabled = true
	return nil
}

func (m *PluginManager) Disable(ctx context.Context, id string) error {
	plugin, ok := m.plugins[id]
	if !ok {
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

func (m *PluginManager) Plugins() []Plugin {
	var plugins []Plugin
	for _, v := range m.plugins {
		plugins = append(plugins, *v)
	}
	return plugins
}

func (m *PluginManager) Runtime(id string) (*PluginRuntime, error) {
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

	if plugin.Error != nil {
		return nil, fmt.Errorf("%w: %s", ErrPluginRuntime, *plugin.Error)
	}

	return nil, ErrPluginRuntime
}
