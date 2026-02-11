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

func TestConfigSaveAndLoadV2(t *testing.T) {
	tempConfigRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfigRoot)

	cfg := &Config{}
	if err := cfg.AddRegistry("/tmp/registry-a"); err != nil {
		t.Fatalf("add local registry failed: %v", err)
	}
	if err := cfg.AddRegistry("https://github.com/acme/skills.git#main"); err != nil {
		t.Fatalf("add git registry failed: %v", err)
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
	if len(loaded.Registries) != 2 {
		t.Fatalf("expected two registries, got %#v", loaded.Registries)
	}

	if loaded.Registries[0].Type != RegistryTypeGit && loaded.Registries[1].Type != RegistryTypeGit {
		t.Fatalf("expected a git registry in loaded config: %#v", loaded.Registries)
	}

	if len(loaded.Harnesses) != 1 || loaded.Harnesses[0] != filepath.Clean("/tmp/harness-a") {
		t.Fatalf("unexpected loaded harnesses: %#v", loaded.Harnesses)
	}
}

func TestLoadLegacyConfigMigratesLocalRegistries(t *testing.T) {
	tempConfigRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfigRoot)

	configPath, err := ConfigPath()
	if err != nil {
		t.Fatalf("config path failed: %v", err)
	}

	legacy := `registries = ["/tmp/registry-a"]
harnesses = ["/tmp/harness-a"]
`

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy config failed: %v", err)
	}

	loaded, _, err := Load()
	if err != nil {
		t.Fatalf("legacy load failed: %v", err)
	}

	if len(loaded.Registries) != 1 {
		t.Fatalf("expected one migrated registry, got %#v", loaded.Registries)
	}
	if loaded.Registries[0].Type != RegistryTypeLocal {
		t.Fatalf("expected local registry type, got %s", loaded.Registries[0].Type)
	}
	if loaded.Registries[0].Source != filepath.Clean("/tmp/registry-a") {
		t.Fatalf("unexpected migrated source: %s", loaded.Registries[0].Source)
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

	cfg.RemoveRegistry(cfg.Registries[0].ID)
	if len(cfg.Registries) != 0 {
		t.Fatalf("expected registry removed, got %#v", cfg.Registries)
	}
}

func TestIsGitSource(t *testing.T) {
	valid := []string{
		"https://github.com/acme/skills",
		"https://github.com/acme/skills.git",
		"git@github.com:acme/skills.git",
		"ssh://git@github.com/acme/skills.git",
	}

	for _, candidate := range valid {
		if !IsGitSource(candidate) {
			t.Fatalf("expected git source to be valid: %s", candidate)
		}
	}

	if IsGitSource("/tmp/local-registry") {
		t.Fatalf("expected local path to not be treated as git source")
	}
}
