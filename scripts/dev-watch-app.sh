#!/usr/bin/env sh
set -eu

go build -o ./tmp/serve ./cmd

exec ./tmp/serve serve \
  -env "$ENV" \
  -db "$PG_URL" \
  -xai-key "$XAI_KEY" \
  -openai-key "$OPENAI_KEY" \
  -pushover-key "$PUSHOVER_KEY" \
  -brave-key "$BRAVE_KEY"
