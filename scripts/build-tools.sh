#!/usr/bin/env sh
# SPDX-License-Identifier: GPL-3.0-only

set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

if [ -n "${ARION_GO_BIN:-}" ] && [ -x "$ARION_GO_BIN" ]; then
  go_binary=$ARION_GO_BIN
elif command -v go >/dev/null 2>&1; then
  go_binary=$(command -v go)
elif [ -n "${HOME:-}" ] && [ -x "$HOME/.local/go/bin/go" ]; then
  go_binary="$HOME/.local/go/bin/go"
elif [ -x /usr/local/go/bin/go ]; then
  go_binary=/usr/local/go/bin/go
else
  printf '%s\n' 'Go 1.22 ou superior não foi encontrado.' >&2
  exit 127
fi

mkdir -p "$project_root/tools"
cd "$project_root"
exec "$go_binary" build -o tools/arion-provider-validator ./cmd/arion-provider-validator
