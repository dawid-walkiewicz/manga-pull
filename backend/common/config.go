package common

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

type ConfigData struct {
	Workers         int  `json:"workers"`
	MaxRetries      int  `json:"maxRetries"`
	IgnoreReuploads bool `json:"ignoreReuploads"`
}

type ConfigManager struct {
	mu       sync.RWMutex
	filePath string
	data     ConfigData
}

func NewConfigManager(path string) (*ConfigManager, error) {
	cm := &ConfigManager{filePath: path}
	if err := cm.Load(); err != nil {
		return nil, err
	}
	return cm, nil
}

func (cm *ConfigManager) Get() ConfigData {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.data
}

func (cm *ConfigManager) UpdateAndSave(newData ConfigData) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	bytes, err := json.MarshalIndent(newData, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := cm.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, bytes, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpFile, cm.filePath); err != nil {
		return err
	}

	cm.data = newData
	return nil
}

func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	defaultConfig := ConfigData{
		Workers:         1,
		MaxRetries:      3,
		IgnoreReuploads: false,
	}

	bytes, err := os.ReadFile(cm.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cm.data = defaultConfig

			return cm.saveUnlocked(cm.data)
		}

		return err
	}
	cm.data = defaultConfig

	return json.Unmarshal(bytes, &cm.data)
}

func (cm *ConfigManager) saveUnlocked(data ConfigData) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := cm.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, bytes, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, cm.filePath)
}
