#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  ./restore-home.sh /path/to/aman-wsl-backup-YYYYMMDD.tar.zst [destination]

Examples:
  ./restore-home.sh ./aman-wsl-backup-20260323.tar.zst /
  ./restore-home.sh ./aman-wsl-backup-20260323.tar.zst /tmp/restore-check

Notes:
  - Use destination "/" to restore into a fresh WSL filesystem.
  - Use another destination first if you want to inspect the archive contents.
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

if (($# < 1 || $# > 2)); then
  usage >&2
  exit 1
fi

ARCHIVE_PATH="$1"
DESTINATION="${2:-/}"

require_command tar "Install GNU tar if it is missing."
require_command zstd "Install it with: sudo apt update && sudo apt install -y zstd"

if [[ ! -f "$ARCHIVE_PATH" ]]; then
  printf 'Archive not found: %s\n' "$ARCHIVE_PATH" >&2
  exit 1
fi

mkdir -p -- "$DESTINATION"

printf 'Restoring %s into %s\n' "$ARCHIVE_PATH" "$DESTINATION"

tar \
  -C "$DESTINATION" \
  --use-compress-program='zstd -d -T0' \
  -xpf "$ARCHIVE_PATH"

printf 'Restore complete.\n'
