package registrysync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"skiller/internal/config"
)

type SyncResult struct {
	RepoPath string
	Output   string
}

type SyncError struct {
	Step   string
	Output string
	Err    error
}

func (e *SyncError) Error() string {
	if strings.TrimSpace(e.Output) == "" {
		return fmt.Sprintf("%s: %v", e.Step, e.Err)
	}
	return fmt.Sprintf("%s: %v (%s)", e.Step, e.Err, strings.TrimSpace(e.Output))
}

func (e *SyncError) Unwrap() error {
	return e.Err
}

func SyncRegistry(registry config.Registry, interactive bool, timeout time.Duration) (SyncResult, error) {
	if !registry.IsRemote() {
		return SyncResult{}, errors.New("registry is not remote")
	}

	repoPath, err := config.RegistryCachePath(registry)
	if err != nil {
		return SyncResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		return SyncResult{}, err
	}

	ctx := context.Background()
	cancel := func() {}
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	if !isGitRepo(repoPath) {
		if err := cloneRepo(ctx, registry, repoPath, interactive); err != nil {
			return SyncResult{RepoPath: repoPath}, err
		}
		return SyncResult{RepoPath: repoPath}, nil
	}

	originURL, err := gitOutput(ctx, repoPath, false, "config", "--get", "remote.origin.url")
	if err != nil {
		return SyncResult{RepoPath: repoPath}, &SyncError{Step: "origin-url", Output: originURL, Err: err}
	}

	if strings.TrimSpace(originURL) != strings.TrimSpace(registry.Source) {
		if err := os.RemoveAll(repoPath); err != nil {
			return SyncResult{RepoPath: repoPath}, err
		}
		if err := cloneRepo(ctx, registry, repoPath, interactive); err != nil {
			return SyncResult{RepoPath: repoPath}, err
		}
		return SyncResult{RepoPath: repoPath}, nil
	}

	if err := fetchAndReset(ctx, registry, repoPath, interactive); err != nil {
		return SyncResult{RepoPath: repoPath}, err
	}

	return SyncResult{RepoPath: repoPath}, nil
}

func RemoveRegistryCache(registry config.Registry) error {
	repoPath, err := config.RegistryCachePath(registry)
	if err != nil {
		return err
	}

	cacheDir := filepath.Dir(repoPath)
	if _, err := os.Stat(cacheDir); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	return os.RemoveAll(cacheDir)
}

func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	markers := []string{
		"terminal prompts disabled",
		"authentication failed",
		"could not read username",
		"permission denied (publickey)",
		"publickey",
		"passphrase",
		"repository not found",
	}

	for _, marker := range markers {
		if strings.Contains(message, marker) {
			return true
		}
	}

	return false
}

func cloneRepo(ctx context.Context, registry config.Registry, repoPath string, interactive bool) error {
	args := []string{"clone", "--depth=1"}
	if strings.TrimSpace(registry.Ref) != "" {
		args = append(args, "--branch", registry.Ref)
	}
	args = append(args, registry.Source, repoPath)

	cloneOutput, err := gitOutput(ctx, "", interactive, args...)
	if err != nil {
		return &SyncError{Step: "clone", Output: cloneOutput, Err: err}
	}

	return nil
}

func fetchAndReset(ctx context.Context, registry config.Registry, repoPath string, interactive bool) error {
	fetchArgs := []string{"fetch", "--depth=1", "origin"}
	if strings.TrimSpace(registry.Ref) != "" {
		fetchArgs = append(fetchArgs, registry.Ref)
	}

	fetchOutput, err := gitOutput(ctx, repoPath, interactive, fetchArgs...)
	if err != nil {
		return &SyncError{Step: "fetch", Output: fetchOutput, Err: err}
	}

	resetOutput, err := gitOutput(ctx, repoPath, interactive, "reset", "--hard", "FETCH_HEAD")
	if err != nil {
		return &SyncError{Step: "reset", Output: resetOutput, Err: err}
	}

	_, _ = gitOutput(ctx, repoPath, interactive, "clean", "-fd")
	return nil
}

func gitOutput(ctx context.Context, dir string, interactive bool, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	cmd.Env = append([]string{}, os.Environ()...)
	if !interactive {
		cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")
	}

	if interactive {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return "", nil
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func isGitRepo(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir()
}
