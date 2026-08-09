#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only

set -euo pipefail

ARION_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ARION_GO_BIN="${ARION_GO_BIN:-/home/kyuubyn/.local/go/bin/go}"
ARION_BACKEND_BIN="$ARION_DIR/backend/arion-backend"
ARION_RUNTIME_PORT="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')"
ARION_RUNTIME_TOKEN="$(python3 -c 'import secrets; print(secrets.token_hex(32))')"
ARION_DESKTOP_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/applications"
ARION_DESKTOP_FILE="$ARION_DESKTOP_DIR/arion.desktop"

mkdir -p "$ARION_DESKTOP_DIR"
ARION_DESKTOP_EXEC="$(printf '%s' "$ARION_DIR/arion-launcher.sh" | sed 's/[&|]/\\&/g')"
ARION_DESKTOP_ICON="$(printf '%s' "$ARION_DIR/assets/icon.png" | sed 's/[&|]/\\&/g')"
sed \
  -e "s|@ARION_EXEC@|$ARION_DESKTOP_EXEC|g" \
  -e "s|@ARION_ICON@|$ARION_DESKTOP_ICON|g" \
  "$ARION_DIR/arion.desktop" > "$ARION_DESKTOP_FILE.tmp"
chmod 0644 "$ARION_DESKTOP_FILE.tmp"
mv "$ARION_DESKTOP_FILE.tmp" "$ARION_DESKTOP_FILE"
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$ARION_DESKTOP_DIR" >/dev/null 2>&1 || true
fi

if [ ! -x "$ARION_GO_BIN" ]; then
  ARION_GO_BIN="$(command -v go)"
fi

if [ ! -f "$ARION_BACKEND_BIN" ] || find "$ARION_DIR/backend" -name '*.go' -newer "$ARION_BACKEND_BIN" -print -quit | grep -q .; then
  echo "[Arion] Compilando a galeria de mídia..."
  "$ARION_GO_BIN" build -o "$ARION_BACKEND_BIN" ./backend
fi

ARION_ELECTRON_BIN="$ARION_DIR/node_modules/.bin/electron"
if [ -x "$ARION_ELECTRON_BIN" ]; then
  # Some automation environments export ELECTRON_RUN_AS_NODE. Arion always
  # needs the real Chromium runtime here.
  env -u ELECTRON_RUN_AS_NODE "$ARION_ELECTRON_BIN" "$ARION_DIR"
  exit $?
fi

ARION_PORT="$ARION_RUNTIME_PORT" ARION_SESSION_TOKEN="$ARION_RUNTIME_TOKEN" "$ARION_BACKEND_BIN" &
ARION_BACKEND_PID=$!

cleanup() {
  curl -fsS -X POST -H "Authorization: Bearer $ARION_RUNTIME_TOKEN" "http://127.0.0.1:$ARION_RUNTIME_PORT/api/shutdown" >/dev/null 2>&1 || true
  kill "$ARION_BACKEND_PID" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

for _ in $(seq 1 40); do
  if curl -fsS "http://127.0.0.1:$ARION_RUNTIME_PORT/api/health" >/dev/null 2>&1; then
    break
  fi
  sleep .1
done

ARION_URL="http://127.0.0.1:$ARION_RUNTIME_PORT/#session=$ARION_RUNTIME_TOKEN"
python3 "$ARION_DIR/arion-window.py" "$ARION_URL"
