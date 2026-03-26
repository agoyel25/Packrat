# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o packrat

# Run directly
go run . backup
go run . restore <archive-path>
go run . inspect <archive-path>

# Run a specific package's tests (if added)
go test ./internal/archive/...
go test ./...
```

## Architecture

**packrat** is a Go CLI tool for creating/restoring compressed backups of WSL home directories. It uses Cobra for CLI structure and Charmbracelet (bubbletea/huh/lipgloss) for its TUI.

### Key flows

**Backup:** `cmd/backup.go` → collects category selections via `internal/tui/wizard.go` → resolves paths via `internal/categories/categories.go` → writes `.tar.zst` via `internal/archive/write.go` → saves preferences to `internal/profile/profile.go`

**Restore:** `cmd/restore.go` → `internal/archive/read.go` (extracts with path sanitization) → optionally restores packages via `internal/packages/packages.go`

**Inspect:** `cmd/inspect.go` → `internal/archive/read.go` reads tar headers only → TUI table via `internal/tui/inspect.go`

### Package summary

| Package | Role |
|---|---|
| `cmd/` | Cobra commands: root, backup, restore, inspect, packages |
| `internal/archive/` | Tar+Zstd read/write with progress callbacks |
| `internal/categories/` | Category definitions (ssh, shell, editors, ai_agents, home_dirs) with built-in prune rules (node_modules, .cache, npm cache, etc.) |
| `internal/profile/` | Persists user choices to `~/.config/packrat/profile.json`; enables `--non-interactive` mode |
| `internal/packages/` | Detects apt/pacman/snap; exports/imports `.packages.json` sidecar |
| `internal/tui/` | Wizard forms, progress bar, completion screen, inspect table; styled with Catppuccin theme |

### Flags

- `--non-interactive` / `-y` on root: skips TUI wizard, uses saved profile
- `--dry-run` on backup: counts files without writing the archive

### Archive format

Output is `.tar.zst` (tar + Zstandard compression via `github.com/klauspost/compress`). Package snapshots are written as a `.packages.json` sidecar alongside the archive.

### Shell wrappers

`backup-home.sh` and `restore-home.sh` are standalone bash scripts that replicate core logic without the Go binary — useful as reference for what the tool does.
