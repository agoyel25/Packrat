# packrat

A Go CLI tool for creating and restoring compressed backups of WSL home directories — with an interactive TUI wizard, smart category detection, and package manager snapshots.

## Features

- Interactive wizard built with [Charmbracelet](https://charm.sh/) (bubbletea + huh + lipgloss)
- Backup by category: SSH keys, shell config, editor settings, AI agent configs, cloud credentials, and more
- Zstandard (`.tar.zst`) compression via `github.com/klauspost/compress`
- Package manager snapshots (apt, pacman, snap) saved as a `.packages.json` sidecar
- Saved profile for `--non-interactive` / scripted runs
- `--dry-run` flag to count files without writing anything
- Path sanitization on restore to prevent directory traversal
- Catppuccin-themed progress bar and completion screen

## Installation

**Prerequisites:** Go 1.22+

```bash
git clone https://github.com/agoyel25/Packrat.git
cd Packrat
go build -o packrat
sudo mv packrat /usr/local/bin/   # optional: put it on PATH
```

## Usage

### Backup

```bash
# Interactive wizard — choose categories, output path, dry-run option
packrat backup

# Write to a specific path
packrat backup ~/backups/my-backup.tar.zst

# Count files without writing the archive
packrat backup --dry-run

# Skip the wizard, use your saved profile
packrat -y backup
```

### Restore

```bash
# Interactive wizard — pick archive and destination
packrat restore

# Restore to a staging directory for inspection
packrat restore ./my-backup.tar.zst /tmp/restore-check

# Restore to the real filesystem (requires root for some files)
sudo packrat restore ./my-backup.tar.zst /

# Use saved profile's last archive path
sudo packrat -y restore
```

### Inspect

```bash
# Browse archive contents in a TUI table
packrat inspect ./my-backup.tar.zst
```

### Packages

```bash
# Restore installed packages from a sidecar snapshot
packrat packages restore ./my-backup.packages.json
```

## Backup categories

| Category | What's included |
|---|---|
| **Home Directories** | All top-level non-hidden dirs in `~` (except `packrat` and `snap`) |
| **SSH** | `~/.ssh` |
| **Shell** | `.bashrc`, `.zshrc`, `.profile`, `.bash_aliases`, `.zshenv`, etc. |
| **AI Agents** | `.claude`, `.codex`, `.cursor`, `.aider`, `.opencode`, `.codeium`, `.tabnine`, and more |
| **Editors** | `.vscode`, `.config/nvim`, `.vim`, `.vimrc`, `.config/helix` |
| **Cloud & DevOps** | `.aws`, `.kube`, `.config/gcloud`, `.docker`, `.terraform.d`, `.azure`, etc. |

### Always pruned

These are excluded regardless of category selection to keep archives lean:

- `node_modules/`, `.cache/` — anywhere in the tree
- `~/.npm`, `~/.bun/install/cache`
- `~/.claude/{cache,image-cache,plugins/cache,debug}`
- `~/.codex/{log,tmp,shell_snapshots}`
- `~/.config/gcloud/{logs,.install}`, `~/.kube/cache`, `~/.aws/sso/cache`

## Profile

After each backup, choices are saved to `~/.config/packrat/profile.json`. Pass `--non-interactive` (or `-y`) to any command to skip the wizard and reuse those saved settings — useful for cron jobs or scripts.

## Archive format

Output files are named `<username>-packrat-YYYYMMDD.tar.zst`. If package export is enabled, a matching `<username>-packrat-YYYYMMDD.packages.json` sidecar is created alongside the archive.

## Shell scripts

`backup-home.sh` and `restore-home.sh` in the repo root are standalone bash scripts that replicate the core backup/restore logic without requiring the Go binary — handy as a reference or fallback.

## Tech stack

| Library | Purpose |
|---|---|
| [cobra](https://github.com/spf13/cobra) | CLI structure |
| [bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework |
| [huh](https://github.com/charmbracelet/huh) | Form/wizard components |
| [lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling (Catppuccin theme) |
| [compress/zstd](https://github.com/klauspost/compress) | Zstandard compression |

## License

MIT
