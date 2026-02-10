package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	AppName        = "skiller"
	ConfigFileName = "config.toml"
)

var knownHarnessCandidates = []string{
	"~/.config/opencode/skills",
	"~/.claude/skills",
	"~/.agents/skills",
}

type Config struct {
	Registries []string `toml:"registries"`
	Harnesses  []string `toml:"harnesses"`
}

func Load() (*Config, string, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, "", err
	}

	cfg := &Config{}
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		return cfg, configPath, nil
	} else if err != nil {
		return nil, "", err
	}

	if _, err := toml.DecodeFile(configPath, cfg); err != nil {
		return nil, "", err
	}

	cfg.Registries = normalizePaths(cfg.Registries)
	cfg.Harnesses = normalizePaths(cfg.Harnesses)

	return cfg, configPath, nil
}

func ConfigPath() (string, error) {
	root, err := configRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, AppName, ConfigFileName), nil
}

func configRoot() (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return ExpandPath(xdg)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

func (c *Config) Save(path string) error {
	c.Registries = dedupePaths(c.Registries)
	c.Harnesses = dedupePaths(c.Harnesses)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(c)
}

func (c *Config) AddRegistry(path string) error {
	normalized, err := ExpandPath(path)
	if err != nil {
		return err
	}
	c.Registries = appendUniquePath(c.Registries, normalized)
	return nil
}

func (c *Config) RemoveRegistry(path string) {
	c.Registries = removePath(c.Registries, path)
}

func (c *Config) AddHarness(path string) error {
	normalized, err := ExpandPath(path)
	if err != nil {
		return err
	}
	c.Harnesses = appendUniquePath(c.Harnesses, normalized)
	return nil
}

func (c *Config) RemoveHarness(path string) {
	c.Harnesses = removePath(c.Harnesses, path)
}

func (c *Config) IsCustomHarness(path string) bool {
	for _, harness := range c.Harnesses {
		if harness == path {
			return true
		}
	}
	return false
}

func DetectKnownHarnesses() []string {
	var found []string
	for _, candidate := range knownHarnessCandidates {
		expanded, err := ExpandPath(candidate)
		if err != nil {
			continue
		}

		info, err := os.Stat(expanded)
		if err != nil {
			continue
		}
		if info.IsDir() {
			found = append(found, filepath.Clean(expanded))
		}
	}
	return dedupePaths(found)
}

func MergeUnique(groups ...[]string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, group := range groups {
		for _, path := range group {
			clean := filepath.Clean(path)
			if clean == "." {
				continue
			}
			if _, ok := seen[clean]; ok {
				continue
			}
			seen[clean] = struct{}{}
			out = append(out, clean)
		}
	}
	return out
}

func ExpandPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path is empty")
	}

	expanded := os.ExpandEnv(trimmed)
	if strings.HasPrefix(expanded, "~/") || expanded == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if expanded == "~" {
			expanded = home
		} else {
			expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
		}
	}

	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}

	return filepath.Clean(abs), nil
}

func normalizePaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		expanded, err := ExpandPath(path)
		if err != nil {
			continue
		}
		normalized = append(normalized, expanded)
	}
	return dedupePaths(normalized)
}

func appendUniquePath(paths []string, path string) []string {
	clean := filepath.Clean(path)
	for _, existing := range paths {
		if filepath.Clean(existing) == clean {
			return paths
		}
	}
	return append(paths, clean)
}

func removePath(paths []string, target string) []string {
	cleanTarget := filepath.Clean(target)
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if filepath.Clean(path) == cleanTarget {
			continue
		}
		out = append(out, path)
	}
	return out
}

func dedupePaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}
