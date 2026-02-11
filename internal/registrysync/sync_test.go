package registrysync

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"skiller/internal/config"
)

func TestIsAuthError(t *testing.T) {
	if !IsAuthError(fmt.Errorf("fatal: could not read Username for 'https://github.com': terminal prompts disabled")) {
		t.Fatalf("expected auth error to be detected")
	}

	if IsAuthError(fmt.Errorf("some generic failure")) {
		t.Fatalf("did not expect generic error to be marked as auth error")
	}
}

func TestRemoveRegistryCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	registry := config.Registry{Type: config.RegistryTypeGit, Source: "https://github.com/acme/skills.git"}
	repoPath, err := config.RegistryCachePath(registry)
	if err != nil {
		t.Fatalf("cache path failed: %v", err)
	}

	cacheDir := filepath.Dir(repoPath)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "placeholder"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if err := RemoveRegistryCache(registry); err != nil {
		t.Fatalf("remove cache failed: %v", err)
	}

	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("expected cache dir to be removed")
	}
}
