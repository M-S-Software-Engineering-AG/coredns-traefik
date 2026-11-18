#!/usr/bin/env bash
# Proves a CoreDNS reload stops the old Traefik pollers: boot polling OLD_URL,
# rewrite the Corefile to NEW_URL, wait for the reload, then OLD_URL must go
# quiet while NEW_URL keeps polling. Needs no Traefik: every tick is logged
# (`debug`) whether or not the API answers.
# Usage: assert-reload.sh <coredns-bin> <corefile-template> [timeout-seconds]
set -euo pipefail

BIN="${1:?coredns binary}"
TEMPLATE="${2:?corefile template}"
TIMEOUT="${3:-60}"
OLD_URL="http://127.0.0.1:8080/api"
NEW_URL="http://localhost:8080/api"

WORK="$(mktemp -d)"
COREFILE="$WORK/Corefile"
LOG="$WORK/coredns.log"
sed "s#__API_URL__#$OLD_URL#" "$TEMPLATE" > "$COREFILE"

"$BIN" -conf "$COREFILE" >"$LOG" 2>&1 &
PID=$!
trap 'kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; rm -rf "$WORK"' EXIT

dump() { echo "----- coredns log -----" >&2; cat "$LOG" >&2; }
sha_count() { grep -c 'Running configuration SHA512' "$LOG" || true; }
# polls_since <line> <url> : refresh ticks for <url> logged after line <line>.
polls_since() { tail -n +"$(($1 + 1))" "$LOG" | grep -cF "[$2] Refreshing sites" || true; }

wait_until() {
  local what="$1" pred="$2" i
  for ((i = 0; i < TIMEOUT; i++)); do
    grep -q 'reload failed' "$LOG" && { echo "reload FAILED" >&2; dump; exit 1; }
    eval "$pred" && { echo "$what after ${i}s"; return 0; }
    sleep 1
  done
  echo "TIMEOUT after ${TIMEOUT}s: $what" >&2; dump; exit 1
}

wait_until "started and polling $OLD_URL" '[ "$(sha_count)" -ge 1 ] && [ "$(polls_since 0 "$OLD_URL")" -ge 1 ]'
FIRST="$(sha_count)"

sed "s#$OLD_URL#$NEW_URL#" "$COREFILE" > "$COREFILE.new" && mv "$COREFILE.new" "$COREFILE"
wait_until "reloaded" '[ "$(sha_count)" -gt "$FIRST" ]'

# Give an in-flight tick of the old poller a moment to land, then watch.
sleep 2
MARK="$(wc -l < "$LOG")"
sleep 5
OLD="$(polls_since "$MARK" "$OLD_URL")"
NEW="$(polls_since "$MARK" "$NEW_URL")"
if [ "$OLD" -gt 0 ]; then
  echo "old poller leaked across reload: $OLD tick(s) for $OLD_URL in 5s" >&2; dump; exit 1
fi
if [ "$NEW" -eq 0 ]; then
  echo "new poller never ran: no ticks for $NEW_URL in 5s" >&2; dump; exit 1
fi
echo "OK: old poller stopped, new poller running ($NEW ticks in 5s)"
