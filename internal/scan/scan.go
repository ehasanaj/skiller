package scan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type Skill struct {
	Name   string
	Path   string
	Parent string
}

func ScanRegistry(registryPath string) ([]Skill, error) {
	cleanRoot := filepath.Clean(registryPath)
	info, err := os.Stat(cleanRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("registry path is not a directory")
	}

	skills := make([]Skill, 0)
	err = filepath.WalkDir(cleanRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if path == cleanRoot {
			return nil
		}

		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !entry.IsDir() {
			return nil
		}

		hasSkill, err := hasSkillMarker(path)
		if err != nil {
			return err
		}
		if !hasSkill {
			return nil
		}

		skills = append(skills, Skill{
			Name:   filepath.Base(path),
			Path:   path,
			Parent: cleanRoot,
		})
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Name == skills[j].Name {
			return skills[i].Path < skills[j].Path
		}
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}

func ScanHarness(harnessPath string) ([]Skill, error) {
	cleanRoot := filepath.Clean(harnessPath)
	info, err := os.Stat(cleanRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Skill{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("harness path is not a directory")
	}

	entries, err := os.ReadDir(cleanRoot)
	if err != nil {
		return nil, err
	}

	skills := make([]Skill, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		skillPath := filepath.Join(cleanRoot, entry.Name())
		hasSkill, err := hasSkillMarker(skillPath)
		if err != nil {
			return nil, err
		}
		if !hasSkill {
			continue
		}

		skills = append(skills, Skill{
			Name:   entry.Name(),
			Path:   skillPath,
			Parent: cleanRoot,
		})
	}

	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, nil
}

func hasSkillMarker(path string) (bool, error) {
	markerPath := filepath.Join(path, "SKILL.md")
	info, err := os.Stat(markerPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}
