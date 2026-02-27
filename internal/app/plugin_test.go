package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPluginManager_LoadExample(t *testing.T) {
	dir := t.TempDir()
	pm, err := NewPluginManager(dir)
	if err != nil {
		t.Fatalf("NewPluginManager: %v", err)
	}

	// Should create example plugin
	plugins := pm.List()
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin (example), got %d", len(plugins))
	}
	if plugins[0].Name != "系統資訊" {
		t.Errorf("example name = %q", plugins[0].Name)
	}
}

func TestPluginManager_LoadCustom(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugins")
	os.MkdirAll(pluginDir, 0755)

	// Write a custom plugin
	yaml := `name: "Test Plugin"
description: "A test plugin"
command: "echo hello"
confirm: true
workspace: true`
	os.WriteFile(filepath.Join(pluginDir, "test.yaml"), []byte(yaml), 0644)

	pm, _ := NewPluginManager(dir)
	plugins := pm.List()

	found := false
	for _, p := range plugins {
		if p.Name == "Test Plugin" {
			found = true
			if p.Command != "echo hello" {
				t.Errorf("command = %q", p.Command)
			}
			if !p.Confirm {
				t.Error("confirm should be true")
			}
			if !p.UseWorkspace {
				t.Error("workspace should be true")
			}
		}
	}
	if !found {
		t.Error("custom plugin not found")
	}
}

func TestPluginManager_Get(t *testing.T) {
	dir := t.TempDir()
	pm, _ := NewPluginManager(dir)

	p, ok := pm.Get(0)
	if !ok {
		t.Error("should get index 0")
	}
	if p.Name == "" {
		t.Error("plugin name should not be empty")
	}

	_, ok = pm.Get(999)
	if ok {
		t.Error("should not get invalid index")
	}
}

func TestPluginManager_Reload(t *testing.T) {
	dir := t.TempDir()
	pm, _ := NewPluginManager(dir)

	initial := pm.Count()

	// Add a new plugin
	yaml := `name: "New Plugin"
description: "Added later"
command: "echo new"`
	os.WriteFile(filepath.Join(dir, "plugins", "new.yaml"), []byte(yaml), 0644)

	pm.Reload()
	if pm.Count() <= initial {
		t.Error("count should increase after reload")
	}
}

func TestPluginManager_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugins")
	os.MkdirAll(pluginDir, 0755)

	// Write invalid YAML
	os.WriteFile(filepath.Join(pluginDir, "bad.yaml"), []byte("{{invalid"), 0644)
	// Write plugin missing required fields
	os.WriteFile(filepath.Join(pluginDir, "empty.yaml"), []byte("description: no name"), 0644)

	pm, _ := NewPluginManager(dir)
	// Should not crash, just skip invalid files
	_ = pm.Count()
}
