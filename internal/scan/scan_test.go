package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRegistryRecursive(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "nested", "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "alpha", "SKILL.md"), []byte("# alpha"), 0o644); err != nil {
		t.Fatalf("write marker failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "nested", "beta"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	skills, err := ScanRegistry(root)
	if err != nil {
		t.Fatalf("scan registry failed: %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "alpha" {
		t.Fatalf("expected skill alpha, got %s", skills[0].Name)
	}
}

func TestScanRegistrySkipsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "source-skill")

	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "SKILL.md"), []byte("# source"), 0o644); err != nil {
		t.Fatalf("write marker failed: %v", err)
	}

	linkPath := filepath.Join(root, "symlinked")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("symlink failed: %v", err)
	}

	skills, err := ScanRegistry(root)
	if err != nil {
		t.Fatalf("scan registry failed: %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill from real folder, got %d", len(skills))
	}
	if skills[0].Path != target {
		t.Fatalf("expected %s, got %s", target, skills[0].Path)
	}
}

func TestScanHarnessListsInstalledSkills(t *testing.T) {
	harness := t.TempDir()

	if err := os.MkdirAll(filepath.Join(harness, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(harness, "alpha", "SKILL.md"), []byte("# alpha"), 0o644); err != nil {
		t.Fatalf("write marker failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(harness, "other"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	skills, err := ScanHarness(harness)
	if err != nil {
		t.Fatalf("scan harness failed: %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("expected 1 installed skill, got %d", len(skills))
	}
	if skills[0].Name != "alpha" {
		t.Fatalf("expected alpha, got %s", skills[0].Name)
	}
}
