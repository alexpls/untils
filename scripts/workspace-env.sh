#!/usr/bin/env sh

# Populate workspace-specific runtime env vars so multiple jj workspaces can run concurrently.

if [ "${UNTILS_WORKSPACE_ENV_LOADED:-0}" = "1" ]; then
	return 0 2>/dev/null || exit 0
fi

workspace_name=""
if command -v jj >/dev/null 2>&1; then
	workspace_name="$(jj --ignore-working-copy workspace list -T 'if(target.current_working_copy(), name, "")' | tr -d '\n')"
fi
if [ -z "${workspace_name}" ]; then
	workspace_name="${WORKSPACE_NAME:-default}"
fi

workspace_slug="$(printf '%s' "${workspace_name}" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')"
if [ -z "${workspace_slug}" ]; then
	workspace_slug="workspace"
fi

workspace_hash="$(printf '%s' "${workspace_name}" | cksum | awk '{print $1}')"
slot=$((workspace_hash % 1000))

if [ "${workspace_name}" = "default" ]; then
	default_app_port="4200"
	default_db_port="54324"
	default_smtp_port="1025"
	default_mailpit_port="8025"
	default_riverui_port="7332"
else
	default_app_port="$((4200 + slot))"
	default_db_port="$((55000 + slot))"
	default_smtp_port="$((11025 + slot))"
	default_mailpit_port="$((18025 + slot))"
	default_riverui_port="$((17332 + slot))"
fi

APP_PORT="${APP_PORT:-$default_app_port}"
DB_PORT="${DB_PORT:-$default_db_port}"
SMTP_PORT="${SMTP_PORT:-$default_smtp_port}"
MAILPIT_PORT="${MAILPIT_PORT:-$default_mailpit_port}"
RIVERUI_PORT="${RIVERUI_PORT:-$default_riverui_port}"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-untils_${workspace_slug}}"
PG_URL="${PG_URL:-postgresql://root:root@localhost:${DB_PORT}/untils_dev}"
PG_TEST_URL="${PG_TEST_URL:-postgresql://root:root@localhost:${DB_PORT}/untils_test}"
SMTP_HOST="${SMTP_HOST:-127.0.0.1}"

export UNTILS_WORKSPACE_ENV_LOADED=1
export WORKSPACE_NAME="${workspace_name}"
export WORKSPACE_SLUG="${workspace_slug}"
export APP_PORT
export DB_PORT
export SMTP_HOST
export SMTP_PORT
export MAILPIT_PORT
export RIVERUI_PORT
export COMPOSE_PROJECT_NAME
export PG_URL
export PG_TEST_URL
