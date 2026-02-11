package config

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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

type RegistryType string

const (
	RegistryTypeLocal RegistryType = "local"
	RegistryTypeGit   RegistryType = "git"
)

type Registry struct {
	ID     string       `toml:"id,omitempty"`
	Name   string       `toml:"name,omitempty"`
	Type   RegistryType `toml:"type"`
	Source string       `toml:"source"`
	Ref    string       `toml:"ref,omitempty"`
	Subdir string       `toml:"subdir,omitempty"`
}

func (r Registry) IsRemote() bool {
	return r.Type == RegistryTypeGit
}

func (r Registry) DisplayName() string {
	if strings.TrimSpace(r.Name) != "" {
		return strings.TrimSpace(r.Name)
	}
	if r.Type == RegistryTypeGit {
		trimmed := strings.TrimSuffix(strings.TrimSpace(r.Source), ".git")
		if idx := strings.LastIndex(trimmed, "/"); idx >= 0 && idx < len(trimmed)-1 {
			return trimmed[idx+1:]
		}
		if idx := strings.LastIndex(trimmed, ":"); idx >= 0 && idx < len(trimmed)-1 {
			return trimmed[idx+1:]
		}
	}
	return strings.TrimSpace(r.Source)
}

type Config struct {
	Registries []Registry `toml:"registries"`
	Harnesses  []string   `toml:"harnesses"`
}

type configV2 struct {
	Registries []Registry `toml:"registries"`
	Harnesses  []string   `toml:"harnesses"`
}

type legacyConfigV1 struct {
	Registries []string `toml:"registries"`
	Harnesses  []string `toml:"harnesses"`
}

func Load() (*Config, string, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, "", err
	}

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		return &Config{}, configPath, nil
	} else if err != nil {
		return nil, "", err
	}

	loadedV2, v2Err := loadV2(configPath)
	if v2Err == nil {
		return loadedV2, configPath, nil
	}

	loadedV1, v1Err := loadLegacyV1(configPath)
	if v1Err == nil {
		return loadedV1, configPath, nil
	}

	return nil, "", v2Err
}

func ConfigPath() (string, error) {
	root, err := configRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, AppName, ConfigFileName), nil
}

func CacheRoot() (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); xdg != "" {
		return ExpandPath(xdg)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".cache"), nil
}

func RegistryCachePath(registry Registry) (string, error) {
	normalized, err := normalizeRegistry(registry)
	if err != nil {
		return "", err
	}

	root, err := CacheRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(root, AppName, "registries", normalized.ID, "repo"), nil
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
	c.Registries = dedupeRegistries(normalizeRegistries(c.Registries))
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

func (c *Config) AddRegistry(input string) error {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return errors.New("path is empty")
	}

	sourceCandidate, ref := splitGitRef(trimmed)
	if IsGitSource(sourceCandidate) {
		return c.AddGitRegistry(sourceCandidate, ref)
	}

	return c.AddLocalRegistry(trimmed)
}

func (c *Config) AddLocalRegistry(path string) error {
	expanded, err := ExpandPath(path)
	if err != nil {
		return err
	}

	registry, err := normalizeRegistry(Registry{
		Type:   RegistryTypeLocal,
		Source: expanded,
	})
	if err != nil {
		return err
	}

	c.Registries = appendUniqueRegistry(c.Registries, registry)
	return nil
}

func (c *Config) AddGitRegistry(source, ref string) error {
	trimmedSource := strings.TrimSpace(source)
	if !IsGitSource(trimmedSource) {
		return errors.New("invalid git registry source")
	}

	registry, err := normalizeRegistry(Registry{
		Type:   RegistryTypeGit,
		Source: trimmedSource,
		Ref:    strings.TrimSpace(ref),
	})
	if err != nil {
		return err
	}

	c.Registries = appendUniqueRegistry(c.Registries, registry)
	return nil
}

func (c *Config) RemoveRegistry(identifier string) {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return
	}

	out := make([]Registry, 0, len(c.Registries))
	for _, registry := range c.Registries {
		if registry.ID == trimmed || registry.Source == trimmed {
			continue
		}
		out = append(out, registry)
	}

	c.Registries = out
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

func IsGitSource(source string) bool {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return false
	}

	if strings.HasPrefix(trimmed, "git@") || strings.HasPrefix(trimmed, "ssh://") || strings.HasPrefix(trimmed, "git://") {
		return true
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return false
	}

	if parsed.Host == "" {
		return false
	}

	switch parsed.Scheme {
	case "https", "http", "ssh":
		return parsed.Path != ""
	default:
		return false
	}
}

func loadV2(configPath string) (*Config, error) {
	decoded := &configV2{}
	if _, err := toml.DecodeFile(configPath, decoded); err != nil {
		return nil, err
	}

	return &Config{
		Registries: dedupeRegistries(normalizeRegistries(decoded.Registries)),
		Harnesses:  normalizePaths(decoded.Harnesses),
	}, nil
}

func loadLegacyV1(configPath string) (*Config, error) {
	decoded := &legacyConfigV1{}
	if _, err := toml.DecodeFile(configPath, decoded); err != nil {
		return nil, err
	}

	cfg := &Config{
		Harnesses: normalizePaths(decoded.Harnesses),
	}

	for _, registryPath := range decoded.Registries {
		if err := cfg.AddLocalRegistry(registryPath); err != nil {
			continue
		}
	}

	cfg.Registries = dedupeRegistries(cfg.Registries)
	return cfg, nil
}

func splitGitRef(input string) (string, string) {
	idx := strings.LastIndex(input, "#")
	if idx <= 0 || idx >= len(input)-1 {
		return input, ""
	}

	return strings.TrimSpace(input[:idx]), strings.TrimSpace(input[idx+1:])
}

func normalizeRegistries(registries []Registry) []Registry {
	out := make([]Registry, 0, len(registries))
	for _, registry := range registries {
		normalized, err := normalizeRegistry(registry)
		if err != nil {
			continue
		}
		out = append(out, normalized)
	}

	return out
}

func normalizeRegistry(registry Registry) (Registry, error) {
	normalized := registry
	normalized.Name = strings.TrimSpace(normalized.Name)
	normalized.Source = strings.TrimSpace(normalized.Source)
	normalized.Ref = strings.TrimSpace(normalized.Ref)
	normalized.Subdir = strings.Trim(strings.TrimSpace(normalized.Subdir), "/")

	if normalized.Type == "" {
		if IsGitSource(normalized.Source) {
			normalized.Type = RegistryTypeGit
		} else {
			normalized.Type = RegistryTypeLocal
		}
	}

	if normalized.Source == "" {
		return Registry{}, errors.New("registry source is empty")
	}

	switch normalized.Type {
	case RegistryTypeLocal:
		expanded, err := ExpandPath(normalized.Source)
		if err != nil {
			return Registry{}, err
		}
		normalized.Source = expanded
		normalized.Ref = ""
	case RegistryTypeGit:
		if !IsGitSource(normalized.Source) {
			return Registry{}, errors.New("invalid git registry source")
		}
	default:
		return Registry{}, errors.New("unsupported registry type")
	}

	if normalized.Subdir != "" {
		normalized.Subdir = filepath.Clean(normalized.Subdir)
		if normalized.Subdir == "." {
			normalized.Subdir = ""
		}
	}

	normalized.ID = strings.TrimSpace(normalized.ID)
	if normalized.ID == "" {
		normalized.ID = registryID(normalized)
	}

	return normalized, nil
}

func registryID(registry Registry) string {
	key := string(registry.Type) + "|" + registry.Source + "|" + registry.Ref + "|" + registry.Subdir
	sum := sha1.Sum([]byte(key))
	encoded := hex.EncodeToString(sum[:])
	if len(encoded) < 12 {
		return encoded
	}
	return encoded[:12]
}

func appendUniqueRegistry(registries []Registry, registry Registry) []Registry {
	for _, existing := range registries {
		if existing.ID == registry.ID {
			return registries
		}
		if existing.Type == registry.Type && existing.Source == registry.Source && existing.Ref == registry.Ref && existing.Subdir == registry.Subdir {
			return registries
		}
	}

	return append(registries, registry)
}

func dedupeRegistries(registries []Registry) []Registry {
	seen := map[string]struct{}{}
	out := make([]Registry, 0, len(registries))

	for _, registry := range registries {
		normalized, err := normalizeRegistry(registry)
		if err != nil {
			continue
		}

		if _, ok := seen[normalized.ID]; ok {
			continue
		}
		seen[normalized.ID] = struct{}{}
		out = append(out, normalized)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Type == out[j].Type {
			if out[i].DisplayName() == out[j].DisplayName() {
				return out[i].Source < out[j].Source
			}
			return out[i].DisplayName() < out[j].DisplayName()
		}
		return out[i].Type < out[j].Type
	})

	return out
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
