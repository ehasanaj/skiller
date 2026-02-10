# skiller

`skiller` is a terminal UI tool for managing AI skill folders across registries and harness paths.

## What it does

- Scans registry directories recursively and lists only folders containing `SKILL.md`.
- Auto-detects known harness skill paths:
  - `~/.config/opencode/skills`
  - `~/.claude/skills`
  - `~/.agents/skills`
- Lets you add and remove custom registries and harness paths.
- Installs skills by copying the full folder (including hidden files) into a target harness path.
- Uninstalls installed skills from harness paths.
- Prompts for confirmation before delete/uninstall operations.

## Build and run

```bash
go build ./cmd/skiller
./skiller
```

## Key bindings

- Navigation: arrow keys or `h/j/k/l`
- Switch panes: `h`/`l` or `tab`
- Add path: `a`
- Delete selected path: `d` (with confirmation)
- Install selected skill to selected harness: `i`
- Uninstall selected installed skill: `u` (with confirmation)
- Rescan: `r`
- Quit: `q`

## Config

Config is stored in:

- `~/.config/skiller/config.toml` (or `$XDG_CONFIG_HOME/skiller/config.toml`)
