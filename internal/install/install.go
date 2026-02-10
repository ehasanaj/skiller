package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"skiller/internal/fsutil"
)

type ConflictAction string

const (
	ConflictSkip      ConflictAction = "skip"
	ConflictOverwrite ConflictAction = "overwrite"
	ConflictRename    ConflictAction = "rename"
)

type InstallResult struct {
	Installed   bool
	Conflict    bool
	Renamed     bool
	Name        string
	Destination string
}

func InstallSkill(skillSourcePath, harnessPath string, action ConflictAction) (InstallResult, error) {
	sourceInfo, err := os.Stat(skillSourcePath)
	if err != nil {
		return InstallResult{}, err
	}
	if !sourceInfo.IsDir() {
		return InstallResult{}, errors.New("skill source path is not a directory")
	}

	if err := os.MkdirAll(harnessPath, 0o755); err != nil {
		return InstallResult{}, err
	}

	skillName := filepath.Base(skillSourcePath)
	destination := filepath.Join(harnessPath, skillName)

	result := InstallResult{
		Name:        skillName,
		Destination: destination,
	}

	if exists(destination) {
		result.Conflict = true
		switch action {
		case ConflictSkip:
			return result, nil
		case ConflictOverwrite:
			if err := os.RemoveAll(destination); err != nil {
				return InstallResult{}, err
			}
		case ConflictRename:
			renamePath, renameName := nextAvailableDestination(harnessPath, skillName)
			destination = renamePath
			result.Name = renameName
			result.Destination = destination
			result.Renamed = true
		default:
			return InstallResult{}, fmt.Errorf("unknown conflict action: %s", action)
		}
	}

	if err := fsutil.CopyDir(skillSourcePath, destination); err != nil {
		return InstallResult{}, err
	}

	result.Installed = true
	return result, nil
}

func UninstallSkill(harnessPath, skillName string) error {
	targetPath := filepath.Join(harnessPath, skillName)
	info, err := os.Stat(targetPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("target path is not a directory")
	}

	markerPath := filepath.Join(targetPath, "SKILL.md")
	markerInfo, err := os.Stat(markerPath)
	if err != nil {
		return fmt.Errorf("target is not a valid installed skill: %w", err)
	}
	if markerInfo.IsDir() {
		return errors.New("invalid skill marker")
	}

	return os.RemoveAll(targetPath)
}

func nextAvailableDestination(harnessPath, skillName string) (string, string) {
	for i := 2; ; i++ {
		candidateName := fmt.Sprintf("%s-%d", skillName, i)
		candidatePath := filepath.Join(harnessPath, candidateName)
		if !exists(candidatePath) {
			return candidatePath, candidateName
		}
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
