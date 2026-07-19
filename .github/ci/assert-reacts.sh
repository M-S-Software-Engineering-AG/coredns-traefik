#!/usr/bin/env bash
# Proves the traefik plugin discovers hosts from a live Traefik API and reacts
# to changes: CoreDNS resolves HOST1 (already advertised), then MUTATE_CMD adds
# a router to Traefik and CoreDNS must start resolving HOST2 with no restart.
# Usage: assert-reacts.sh <coredns-bin> <corefile> <host1> <expected-ip> <mutate-cmd> <host2>
set -euo pipefail

BIN="${1:?coredns binary}"
COREFILE="${2:?corefile}"
HOST1="${3:?host advertised at startup}"
IP="${4:?expected A record}"
MUTATE="${5:?command that changes Traefik config}"
HOST2="${6:?host advertised after mutation}"
TIMEOUT="${TIMEOUT:-60}"
PORT="${DNS_PORT:-5354}"

resolves() { dig @127.0.0.1 -p "$PORT" +short "$1" A | grep -qx "$IP"; }

LOG="$(mktemp)"
"$BIN" -conf "$COREFILE" >"$LOG" 2>&1 &
PID=$!
trap 'kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true' EXIT

wait_for() {
  local host="$1" i
  for ((i = 0; i < TIMEOUT; i++)); do
    resolves "$host" && { echo "$host -> $IP after ${i}s"; return 0; }
    sleep 1
  done
  echo "TIMEOUT after ${TIMEOUT}s: $host never resolved to $IP" >&2
  echo "----- coredns log -----" >&2
  cat "$LOG" >&2
  return 1
}

# 1. Host advertised by Traefik at startup is discovered and served.
wait_for "$HOST1"

# 2. Change Traefik live; the plugin must pick it up on its next poll.
echo "mutating Traefik: $MUTATE"
eval "$MUTATE"
wait_for "$HOST2"

echo "OK: plugin reacted to live Traefik change"
