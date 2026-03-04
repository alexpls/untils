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
	APP_PORT="4200"
	DB_PORT="54324"
	SMTP_PORT="1025"
	MAILPIT_PORT="8025"
	RIVERUI_PORT="7332"
else
	APP_PORT="$((4200 + slot))"
	DB_PORT="$((55000 + slot))"
	SMTP_PORT="$((11025 + slot))"
	MAILPIT_PORT="$((18025 + slot))"
	RIVERUI_PORT="$((17332 + slot))"
fi

COMPOSE_PROJECT_NAME="untils_${workspace_slug}"
PG_URL="postgresql://root:root@localhost:${DB_PORT}/untils_dev"
PG_TEST_URL="postgresql://root:root@localhost:${DB_PORT}/untils_test"
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
