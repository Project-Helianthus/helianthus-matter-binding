#!/usr/bin/env bash
# Run a command in one owned process group and reap every descendant on exit.
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 TIMEOUT_SECONDS COMMAND [ARG...]" >&2
  exit 2
fi

timeout_seconds=$1
shift
if [[ ! "$timeout_seconds" =~ ^[1-9][0-9]*$ ]]; then
  echo "TIMEOUT_SECONDS must be a positive integer" >&2
  exit 2
fi

group_pid=""

stop_group() {
  if [[ -z "$group_pid" ]]; then
    return
  fi
  kill -TERM -- "-$group_pid" 2>/dev/null || true
  for _ in $(seq 1 20); do
    if ! kill -0 -- "-$group_pid" 2>/dev/null; then
      break
    fi
    sleep 0.1
  done
  kill -KILL -- "-$group_pid" 2>/dev/null || true
  wait "$group_pid" 2>/dev/null || true
  group_pid=""
}

trap stop_group EXIT INT TERM
setsid timeout --kill-after=10s "${timeout_seconds}s" "$@" &
group_pid=$!
set +e
wait "$group_pid"
status=$?
set -e
stop_group
exit "$status"
