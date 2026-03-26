#!/usr/bin/env bash

set -euo pipefail

HOME_ROOT="/home/aman"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
DATE_STAMP="$(date +%Y%m%d)"
DEFAULT_ARCHIVE="${SCRIPT_DIR}/aman-wsl-backup-${DATE_STAMP}.tar.zst"
DRY_RUN=0
ARCHIVE_PATH="$DEFAULT_ARCHIVE"

usage() {
  cat <<'EOF'
Usage:
  ./backup-home.sh [--dry-run] [archive-path]

Examples:
  ./backup-home.sh
  ./backup-home.sh --dry-run
  ./backup-home.sh /tmp/aman-wsl-backup-20260323.tar.zst

What gets included:
  - All top-level non-hidden directories in /home/aman except excluded ones like wsl-backup/snap
  - Key hidden config/data paths:
    .agents .claude .codex .ssh
    .gitconfig .claude.json .claude.json.backup

What gets pruned:
  - Any node_modules directory
  - Generic .cache directories
  - /home/aman/.npm
  - /home/aman/.bun/install/cache
  - /home/aman/.claude/cache
  - /home/aman/.claude/image-cache
  - /home/aman/.claude/plugins/cache
  - /home/aman/.claude/debug
  - /home/aman/.codex/log
  - /home/aman/.codex/tmp
  - /home/aman/.codex/shell_snapshots
EOF
}

require_command() {
  local name="$1"
  local install_hint="$2"
  if ! command -v "$name" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "$name" >&2
    printf '%s\n' "$install_hint" >&2
    exit 1
  fi
}

while (($# > 0)); do
  case "$1" in
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      ARCHIVE_PATH="$1"
      shift
      ;;
  esac
done

require_command tar "Install GNU tar if it is missing."
if (( ! DRY_RUN )); then
  require_command zstd "Install it with: sudo apt update && sudo apt install -y zstd"
fi

mkdir -p -- "$(dirname -- "$ARCHIVE_PATH")"
ARCHIVE_PATH="$(cd -- "$(dirname -- "$ARCHIVE_PATH")" && pwd)/$(basename -- "$ARCHIVE_PATH")"
MANIFEST_PATH="${ARCHIVE_PATH%.tar.zst}.manifest.txt"

declare -a include_roots=()
declare -a hidden_roots=(
  "home/aman/.agents"
  "home/aman/.claude"
  "home/aman/.codex"
  "home/aman/.ssh"
  "home/aman/.gitconfig"
  "home/aman/.claude.json"
  "home/aman/.claude.json.backup"
)

declare -a top_level_excludes=(
  "wsl-backup"
  "snap"
)

for path in "${HOME_ROOT}"/*; do
  [[ -d "$path" ]] || continue
  base_name="$(basename -- "$path")"
  skip=0
  for excluded in "${top_level_excludes[@]}"; do
    if [[ "$base_name" == "$excluded" ]]; then
      skip=1
      break
    fi
  done
  ((skip)) && continue
  include_roots+=("home/aman/${base_name}")
done

for hidden_path in "${hidden_roots[@]}"; do
  if [[ -e "/${hidden_path}" ]]; then
    include_roots+=("$hidden_path")
  fi
done

if ((${#include_roots[@]} == 0)); then
  printf 'Nothing matched for backup under %s\n' "$HOME_ROOT" >&2
  exit 1
fi

declare -a prune_dirs=(
  "home/aman/.npm"
  "home/aman/.bun/install/cache"
  "home/aman/.claude/cache"
  "home/aman/.claude/image-cache"
  "home/aman/.claude/plugins/cache"
  "home/aman/.claude/debug"
  "home/aman/.codex/log"
  "home/aman/.codex/tmp"
  "home/aman/.codex/shell_snapshots"
)

temp_dir="$(mktemp -d)"
file_list="${temp_dir}/backup-files.list"
trap 'rm -rf -- "$temp_dir"' EXIT

declare -a find_cmd=(
  find
)

find_cmd+=("${include_roots[@]}")
find_cmd+=(
  "("
  "-type"
  "d"
  "("
  "-name"
  "node_modules"
  "-o"
  "-name"
  ".cache"
)

for prune_dir in "${prune_dirs[@]}"; do
  find_cmd+=("-o" "-path" "$prune_dir")
done

find_cmd+=(
  ")"
  "-prune"
  ")"
  "-o"
  "-print0"
)

(
  cd /
  "${find_cmd[@]}"
) >"$file_list"

file_count="$(tr '\0' '\n' <"$file_list" | wc -l | awk '{print $1}')"

{
  printf 'Archive: %s\n' "$ARCHIVE_PATH"
  printf 'Created: %s\n' "$(date -u '+%Y-%m-%d %H:%M:%S UTC')"
  printf 'Home root: %s\n' "$HOME_ROOT"
  printf 'Entries in archive file list: %s\n' "$file_count"
  printf '\nIncluded top-level roots:\n'
  printf '  %s\n' "${include_roots[@]}"
  printf '\nPruned directories/patterns:\n'
  printf '  %s\n' "any node_modules directory"
  printf '  %s\n' "any .cache directory"
  printf '  %s\n' "${prune_dirs[@]}"
} >"$MANIFEST_PATH"

printf 'Archive target: %s\n' "$ARCHIVE_PATH"
printf 'Manifest: %s\n' "$MANIFEST_PATH"
printf 'Roots selected: %s\n' "${#include_roots[@]}"
printf 'Entries selected: %s\n' "$file_count"

if ((DRY_RUN)); then
  printf 'Dry run complete. No archive was created.\n'
  exit 0
fi

tar \
  -C / \
  --use-compress-program='zstd -T0 -10' \
  -cpf "$ARCHIVE_PATH" \
  --null \
  -T "$file_list"

printf 'Backup complete: %s\n' "$ARCHIVE_PATH"
