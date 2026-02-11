# skiller

`skiller` is a terminal UI (TUI) for managing AI skill folders across multiple registries and harnesses.

It helps you discover skills, install them into harness-specific directories, and keep skill sources organized.

## Features

- Recursively scans registry directories and lists only valid skill folders (folders containing `SKILL.md`).
- Supports registry sources from:
  - local filesystem paths
  - remote git repositories (GitHub/Git-compatible) via HTTPS or SSH
- Auto-detects popular harness skill directories:
  - `~/.config/opencode/skills`
  - `~/.claude/skills`
  - `~/.agents/skills`
- Supports adding and removing custom registries and custom harness paths.
- Caches remote registries locally and scans the cache.
- Runs non-interactive remote sync on startup, plus manual sync from the UI.
- Installs a skill by copying the full folder (including hidden files) into a harness path.
- Preserves file permissions while copying.
- Handles install conflicts with actions: `overwrite`, `rename`, or `skip`.
- Lists installed skills grouped by harness.
- Supports uninstall with confirmation.
- Keyboard-first UX with Vim-style and arrow-key navigation.

## Core Concepts

- `Skill`: any folder that contains a `SKILL.md` file.
- `Registry`: a source directory where skills are discovered.
  - can be a local path or a remote git repository
- `Harness path`: a destination directory where skills are installed.

## Installation

### Prerequisites

- Go `1.22+`

### Build from source

```bash
git clone <your-repo-url>
cd skiller
go build -o skiller ./cmd/skiller
```

### Install binary to PATH (macOS/Linux)

```bash
mkdir -p "$HOME/.local/bin"
cp ./skiller "$HOME/.local/bin/skiller"
chmod +x "$HOME/.local/bin/skiller"
```

Make sure `$HOME/.local/bin` is in your `PATH`.

### Run without installing globally

```bash
go run ./cmd/skiller
```

## Quick Start

1. Launch `skiller`.
2. In the `Registries` pane, press `a` to add a local path or git URL.
   - Optional branch/tag syntax: `https://github.com/org/repo.git#main`
3. If you added a remote registry, press `s` to sync selected registry (or `S` for all remotes).
4. In the `Harness Installs` pane, select an auto-detected harness or add one with `a`.
5. In `Registry Skills`, pick a skill and press `i` to install.
6. To uninstall, select an installed skill in `Harness Installs` and press `u`.

## TUI Layout

`skiller` uses a fullscreen 3-pane dashboard:

- `Registries` (left): configured local and remote registries.
- `Registry Skills` (middle): skills found in selected registry.
- `Harness Installs` (right): installed skills grouped by harness path.

The currently focused pane is visually highlighted.

## Keybindings

- `up/down` or `k/j`: move selection
- `left/right` or `h/l`: switch pane
- `tab` / `shift+tab`: cycle pane focus
- `a`: add path (registry or harness, depending on focused pane)
- `d`: delete selected path (confirmation required)
- `i`: install selected skill into selected harness
- `u`: uninstall selected installed skill (confirmation required)
- `s`: sync selected remote registry
- `S`: sync all remote registries
- `r`: rescan registries and harnesses
- `q` or `ctrl+c`: quit

### Conflict prompt during install

If destination skill folder already exists:

- `o`: overwrite
- `r`: install with auto-generated renamed folder (`name-2`, `name-3`, ...)
- `s` or `esc`: skip

## Configuration

Config is stored at:

- `$XDG_CONFIG_HOME/skiller/config.toml`, or
- `~/.config/skiller/config.toml` when `XDG_CONFIG_HOME` is not set

Example:

```toml
[[registries]]
id = "a1b2c3d4e5f6"
type = "local"
source = "/Users/alice/skills-registry"

[[registries]]
id = "f6e5d4c3b2a1"
type = "git"
source = "git@github.com:acme/team-skills.git"
ref = "main"

harnesses = [
  "/Users/alice/.my-harness/skills"
]
```

Legacy configs that used `registries = ["/path"]` are migrated automatically on load.

Notes:

- Auto-detected harness paths are added at runtime if they exist.
- Custom harness paths are persisted in config.
- Auto-detected harnesses are not removed via `d`.

## Behavior and Safety Rules

- Registry scanning is recursive.
- Symlinked directories are not traversed during scanning.
- Only directories containing `SKILL.md` are treated as skills.
- Remote registries are scanned from local cache.
- Startup sync is non-interactive (`GIT_TERMINAL_PROMPT=0`) to avoid TUI blocking.
- Manual sync can prompt for SSH passphrase or HTTPS credentials via git.
- Install copies the full directory tree, including dotfiles.
- Delete/uninstall actions require explicit Y/N confirmation.
- Uninstall only removes directories that look like valid skills (must include `SKILL.md`).

## Development

### Project structure

```text
cmd/skiller/            # app entrypoint
internal/config/        # config load/save, path handling, autodetect harnesses
internal/registrysync/  # remote git registry cache sync
internal/scan/          # registry/harness scanning and skill discovery
internal/fsutil/        # filesystem copy helpers
internal/install/       # install/uninstall logic and conflict handling
internal/ui/            # Bubble Tea TUI model and rendering
```

### Useful commands

```bash
go mod tidy
gofmt -w ./...
go test ./...
go run ./cmd/skiller
go build -o skiller ./cmd/skiller
```

## Contributing

Contributions are welcome.

- Open an issue for bugs or feature requests.
- Submit small, focused pull requests.
- Include tests for behavior changes in scan/install/config logic.
- Keep UX changes consistent with keyboard-first operation and always-visible shortcuts.

## Open Source Readiness Checklist

For a public release, recommended next steps:

- Add a `LICENSE` file.
- Add CI for `gofmt` and `go test`.
- Add release automation for cross-platform binaries.
- Add screenshots or an asciinema demo.
