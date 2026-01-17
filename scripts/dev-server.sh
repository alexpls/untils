#!/usr/bin/env bash
set -euo pipefail
set -m

# Workaround for templ watcher not forwarding SIGINT/SIGTERM to the child process
# group, so graceful shutdown is skipped. Required until
# https://github.com/a-h/templ/issues/1323 is fixed.

cmd=(go run ./cmd serve -env "$ENV" -db "$PG_URL" -xai-key "$XAI_KEY" -openai-key "$OPENAI_KEY" -pushover-key "$PUSHOVER_KEY" -brave-key "$BRAVE_KEY")

cleanup() {
  trap - INT TERM EXIT  # Prevent re-entry
  if [[ -n "${child_pid:-}" ]]; then
    kill -TERM -- -"$(ps -o pgid= "$child_pid" | tr -d ' ')" 2>/dev/null || true
    wait "$child_pid" 2>/dev/null || true
  fi
}

trap cleanup INT TERM EXIT

"${cmd[@]}" &
child_pid=$!

wait "$child_pid"
