package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallCopiesEntireSkillFolder(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "alpha")
	harness := filepath.Join(root, "harness")

	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("# alpha"), 0o644); err != nil {
		t.Fatalf("write marker failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, ".hidden"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write hidden file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "run.sh"), []byte("#!/bin/sh"), 0o750); err != nil {
		t.Fatalf("write nested file failed: %v", err)
	}

	result, err := InstallSkill(source, harness, ConflictSkip)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if !result.Installed {
		t.Fatalf("expected install to succeed")
	}

	if _, err := os.Stat(filepath.Join(harness, "alpha", "SKILL.md")); err != nil {
		t.Fatalf("expected copied marker file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(harness, "alpha", ".hidden")); err != nil {
		t.Fatalf("expected copied hidden file: %v", err)
	}

	info, err := os.Stat(filepath.Join(harness, "alpha", "nested", "run.sh"))
	if err != nil {
		t.Fatalf("expected copied executable file: %v", err)
	}
	if info.Mode().Perm() != 0o750 {
		t.Fatalf("expected permissions 0750, got %o", info.Mode().Perm())
	}
}

func TestInstallConflictHandling(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "alpha")
	harness := filepath.Join(root, "harness")

	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("# alpha"), 0o644); err != nil {
		t.Fatalf("write marker failed: %v", err)
	}

	if _, err := InstallSkill(source, harness, ConflictSkip); err != nil {
		t.Fatalf("initial install failed: %v", err)
	}

	result, err := InstallSkill(source, harness, ConflictSkip)
	if err != nil {
		t.Fatalf("skip conflict install failed: %v", err)
	}
	if !result.Conflict || result.Installed {
		t.Fatalf("expected conflict skip to return conflict without install")
	}

	renameResult, err := InstallSkill(source, harness, ConflictRename)
	if err != nil {
		t.Fatalf("rename conflict install failed: %v", err)
	}
	if !renameResult.Installed || !renameResult.Renamed {
		t.Fatalf("expected renamed install")
	}
	if renameResult.Name != "alpha-2" {
		t.Fatalf("expected renamed folder alpha-2, got %s", renameResult.Name)
	}
}

func TestUninstallSkillRequiresMarker(t *testing.T) {
	root := t.TempDir()
	harness := filepath.Join(root, "harness")
	skill := filepath.Join(harness, "alpha")

	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	if err := UninstallSkill(harness, "alpha"); err == nil {
		t.Fatalf("expected uninstall to fail without SKILL.md")
	}

	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("# alpha"), 0o644); err != nil {
		t.Fatalf("write marker failed: %v", err)
	}

	if err := UninstallSkill(harness, "alpha"); err != nil {
		t.Fatalf("expected uninstall to succeed with marker: %v", err)
	}

	if _, err := os.Stat(skill); !os.IsNotExist(err) {
		t.Fatalf("expected skill directory removed")
	}
}
