package plugins

import (
	"context"
	"fmt"
	"main/db"
)

type PluginManager struct {
	store *db.Store
	dir   string

	plugins  []Plugin
	runtimes *RuntimeRegistry
}

func NewPluginManager(store *db.Store, dir string) *PluginManager {
	return &PluginManager{
		store:    store,
		dir:      dir,
		plugins:  make([]Plugin, 0),
		runtimes: NewRuntimeRegistry(),
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

func (m *PluginManager) findPlugin(id string) *Plugin {
	for i := range m.plugins {
		if m.plugins[i].ID == id {
			return &m.plugins[i]
		}
	}

	return nil
}

func (m *PluginManager) Plugin(id string) (*Plugin, bool) {
	plugin := m.findPlugin(id)
	return plugin, plugin != nil
}

func applyPluginState(
	discovered []Plugin,
	records []db.PluginRecord,
) []Plugin {
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
	paths, err := FindPluginZips(m.dir)
	if err != nil {
		return fmt.Errorf("find plugins: %w", err)
	}

	discovered := LoadPlugins(paths)
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

		runtime, err := NewRuntime(plugin)
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

	runtime, err := NewRuntime(plugin)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPluginRuntime, err)
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

func (m *PluginManager) Plugins() []Plugin {
	return m.plugins
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

	return nil, ErrPluginRuntime
}
