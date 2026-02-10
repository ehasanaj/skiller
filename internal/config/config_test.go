package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("home dir unavailable: %v", err)
	}

	expanded, err := ExpandPath("~/tmp/skiller")
	if err != nil {
		t.Fatalf("expand path failed: %v", err)
	}

	if !strings.HasPrefix(expanded, home) {
		t.Fatalf("expected expanded path to start with home dir %s, got %s", home, expanded)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tempConfigRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfigRoot)

	cfg := &Config{}
	if err := cfg.AddRegistry("/tmp/registry-a"); err != nil {
		t.Fatalf("add registry failed: %v", err)
	}
	if err := cfg.AddHarness("/tmp/harness-a"); err != nil {
		t.Fatalf("add harness failed: %v", err)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("config path failed: %v", err)
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, loadedPath, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loadedPath != path {
		t.Fatalf("expected path %s, got %s", path, loadedPath)
	}
	if len(loaded.Registries) != 1 || loaded.Registries[0] != filepath.Clean("/tmp/registry-a") {
		t.Fatalf("unexpected loaded registries: %#v", loaded.Registries)
	}
	if len(loaded.Harnesses) != 1 || loaded.Harnesses[0] != filepath.Clean("/tmp/harness-a") {
		t.Fatalf("unexpected loaded harnesses: %#v", loaded.Harnesses)
	}
}

func TestConfigAddRemoveDedupes(t *testing.T) {
	cfg := &Config{}

	if err := cfg.AddRegistry("/tmp/registry-a"); err != nil {
		t.Fatalf("add registry failed: %v", err)
	}
	if err := cfg.AddRegistry("/tmp/registry-a"); err != nil {
		t.Fatalf("duplicate add failed: %v", err)
	}

	if len(cfg.Registries) != 1 {
		t.Fatalf("expected deduped registries, got %#v", cfg.Registries)
	}

	cfg.RemoveRegistry("/tmp/registry-a")
	if len(cfg.Registries) != 0 {
		t.Fatalf("expected registry removed, got %#v", cfg.Registries)
	}
}
