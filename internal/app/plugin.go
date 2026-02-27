package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Plugin represents a user-defined skill loaded from a YAML config file.
type Plugin struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Command     string `yaml:"command" json:"command"`
	Confirm     bool   `yaml:"confirm" json:"confirm"`     // require confirmation before exec
	UseWorkspace bool  `yaml:"workspace" json:"workspace"` // run in current workspace
}

// PluginManager loads and manages user-defined plugins.
type PluginManager struct {
	mu      sync.RWMutex
	dir     string
	plugins []Plugin
}

// NewPluginManager creates a PluginManager that loads from the given directory.
func NewPluginManager(axleDir string) (*PluginManager, error) {
	dir := filepath.Join(axleDir, "plugins")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("建立插件目錄失敗: %w", err)
	}

	pm := &PluginManager{dir: dir}
	if err := pm.Reload(); err != nil {
		slog.Warn("載入插件時有警告", "error", err)
	}

	// Create example plugin if directory is empty
	entries, _ := os.ReadDir(dir)
	if len(entries) == 0 {
		pm.createExample()
		_ = pm.Reload() // reload to include the example
	}

	return pm, nil
}

// Reload reads all YAML/JSON plugin files from the plugin directory.
func (pm *PluginManager) Reload() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.plugins = nil

	entries, err := os.ReadDir(pm.dir)
	if err != nil {
		return fmt.Errorf("讀取插件目錄失敗: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		path := filepath.Join(pm.dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("讀取插件失敗", "file", entry.Name(), "error", err)
			continue
		}

		var plugin Plugin
		switch ext {
		case ".json":
			err = json.Unmarshal(data, &plugin)
		default:
			err = yaml.Unmarshal(data, &plugin)
		}
		if err != nil {
			slog.Warn("解析插件失敗", "file", entry.Name(), "error", err)
			continue
		}

		if plugin.Name == "" || plugin.Command == "" {
			slog.Warn("插件缺少必要欄位", "file", entry.Name())
			continue
		}

		pm.plugins = append(pm.plugins, plugin)
	}

	slog.Info("✅ 插件已載入", "count", len(pm.plugins))
	return nil
}

// List returns all loaded plugins.
func (pm *PluginManager) List() []Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]Plugin, len(pm.plugins))
	copy(result, pm.plugins)
	return result
}

// Get returns a plugin by index.
func (pm *PluginManager) Get(index int) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if index < 0 || index >= len(pm.plugins) {
		return Plugin{}, false
	}
	return pm.plugins[index], true
}

// Count returns the number of loaded plugins.
func (pm *PluginManager) Count() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.plugins)
}

func (pm *PluginManager) createExample() {
	example := `# Axle Plugin Example
# 將此檔案放在 ~/.axle/plugins/ 目錄下
name: "系統資訊"
description: "顯示系統基本資訊 (uname, uptime, disk)"
command: "echo '=== System ===' && uname -a && echo '=== Uptime ===' && uptime && echo '=== Disk ===' && df -h /"
confirm: false
workspace: false
`
	path := filepath.Join(pm.dir, "example.yaml")
	if err := os.WriteFile(path, []byte(example), 0644); err != nil {
		slog.Warn("建立範例插件失敗", "error", err)
	}
}
