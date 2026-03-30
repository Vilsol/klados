#!/usr/bin/env bash
# Installs hello-plugin into the klados plugins directory.
# Run from any directory. Requires that the plugin has been built first
# (npm install && npm run build).
set -euo pipefail

PLUGIN_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/klados/plugins/hello-plugin"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ ! -f "$SCRIPT_DIR/ui/HelloTab.js" ]; then
  echo "Error: ui/HelloTab.js not found. Build first: npm install && npm run build" >&2
  exit 1
fi

mkdir -p "$PLUGIN_DIR/ui"
cp "$SCRIPT_DIR/manifest.json" "$PLUGIN_DIR/"
cp "$SCRIPT_DIR/ui/HelloTab.js" "$PLUGIN_DIR/ui/"
[ -f "$SCRIPT_DIR/ui/HelloTab.js.map" ] && cp "$SCRIPT_DIR/ui/HelloTab.js.map" "$PLUGIN_DIR/ui/"

echo "Installed to: $PLUGIN_DIR"
echo "Restart klados (or wait for hot-reload) to activate the plugin."
