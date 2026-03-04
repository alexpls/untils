#!/usr/bin/env sh
set -eu

. ./scripts/workspace-env.sh

go build -o ./tmp/serve ./cmd

exec ./tmp/serve serve \
  -port "$APP_PORT" \
  -env "$ENV" \
  -db "$PG_URL" \
  -smtp-host "$SMTP_HOST" \
  -smtp-port "$SMTP_PORT" \
  -xai-key "$XAI_KEY" \
  -openai-key "$OPENAI_KEY" \
  -pushover-key "$PUSHOVER_KEY" \
  -brave-key "$BRAVE_KEY"
