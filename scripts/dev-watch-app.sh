#!/usr/bin/env sh
set -eu

. ./scripts/workspace-env.sh

ENV="${ENV:-dev}"
APP_MODE="${APP_MODE:-hosted}"
BASE_URL="${BASE_URL:-http://untils.localhost:$APP_PORT}"
DEMO_USER_ID="${DEMO_USER_ID:-1}"
SMTP_FROM="${SMTP_FROM:-notifications@untils.com}"

export ENV
export APP_MODE
export BASE_URL
export DEMO_USER_ID
export SMTP_FROM

go generate ./internal/docs
go build -o ./tmp/serve ./cmd

exec ./tmp/serve serve
