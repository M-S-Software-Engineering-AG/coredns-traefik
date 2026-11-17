#!/usr/bin/env bash
# Drops this plugin into a CoreDNS checkout and prepares it for build/test.
# Usage: setup-coredns.sh <coredns-checkout-dir>
set -euo pipefail

COREDNS_DIR="${1:?usage: setup-coredns.sh <coredns-checkout-dir>}"
PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Drop the plugin in (skip its own git dir and the CI harness under .github).
mkdir -p "$COREDNS_DIR/plugin/traefik"
rsync -a --delete --exclude '.git' --exclude '.github' \
  "$PLUGIN_DIR/" "$COREDNS_DIR/plugin/traefik/"

# Register the plugin so `go generate` wires it into CoreDNS. It must be ordered
# BEFORE `file`/`forward`: CoreDNS chains plugins in plugin.cfg order, and this
# plugin serves the hosts it discovers and passes everything else on — so it has
# to run before whatever answers the zone.
if ! grep -q '^traefik:' "$COREDNS_DIR/plugin.cfg"; then
  awk '/^file:file/ && !ins {print "traefik:traefik"; ins=1} {print}' \
    "$COREDNS_DIR/plugin.cfg" > "$COREDNS_DIR/plugin.cfg.tmp"
  mv "$COREDNS_DIR/plugin.cfg.tmp" "$COREDNS_DIR/plugin.cfg"
fi

# Pin the modules listed in deps.ini (CoreDNS's go.mod doesn't require them, so `go mod tidy`
# would otherwise pull whatever is latest).
pins=()
while read -r pin; do pins+=("$pin"); done < <("$PLUGIN_DIR/.github/ci/deps.sh")

cd "$COREDNS_DIR"
(( ${#pins[@]} == 0 )) || go get "${pins[@]}"
go generate
go mod tidy

# tidy keeps whatever the plugin imports. If a major bump in deps.ini isn't matched by the import
# paths, tidy silently falls back to the old major, so fail here instead.
for pin in ${pins[@]+"${pins[@]}"}; do
  got="$(go list -m -f '{{.Version}}' "${pin%@*}" 2>/dev/null || true)"
  [[ "$got" == "${pin#*@}" ]] || { echo "deps.ini pins $pin but the build uses '${got:-nothing}' - are the plugin's imports on that major?" >&2; exit 1; }
done
