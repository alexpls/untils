#!/usr/bin/env bash
set -euo pipefail

generated_start="# --- workspace overrides (generated) ---"
generated_end="# --- end workspace overrides ---"

workspace_name="${1:-}"
source_env_file="${2:-.env.dev}"
dest_env_file="${3:-$source_env_file}"
test_env_file="${TEST_ENV_FILE:-.env.test}"

if [ -z "${workspace_name}" ]; then
	if command -v jj >/dev/null 2>&1; then
		workspace_name="$(jj --ignore-working-copy workspace list -T 'if(target.current_working_copy(), name, "")' | tr -d '\n')"
	fi
fi
if [ -z "${workspace_name}" ]; then
	workspace_name="default"
fi

if [ ! -f "${source_env_file}" ]; then
	echo "missing source env file: ${source_env_file}" >&2
	exit 1
fi

workspace_slug="$(printf '%s' "${workspace_name}" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')"
if [ -z "${workspace_slug}" ]; then
	workspace_slug="workspace"
fi

workspace_hash="$(printf '%s' "${workspace_name}" | cksum | awk '{print $1}')"
slot=$((workspace_hash % 1000))

if [ "${workspace_name}" = "default" ]; then
	app_port="4200"
	db_port="54324"
	smtp_port="1025"
	mailpit_port="8025"
	riverui_port="7332"
	templ_proxy_port="7331"
else
	app_port="$((4200 + slot))"
	db_port="$((55000 + slot))"
	smtp_port="$((11025 + slot))"
	mailpit_port="$((18025 + slot))"
	riverui_port="$((17332 + slot))"
	templ_proxy_port="$((7331 + slot))"
fi

tmp_file="$(mktemp "${TMPDIR:-/tmp}/untils-env-dev.XXXXXX")"
cleanup() {
	rm -f "${tmp_file}"
}
trap cleanup EXIT

skip_generated_block=0
while IFS= read -r line || [ -n "${line}" ]; do
	if [ "${line}" = "${generated_start}" ]; then
		skip_generated_block=1
		continue
	fi
	if [ "${line}" = "${generated_end}" ]; then
		skip_generated_block=0
		continue
	fi
	if [ "${skip_generated_block}" = "1" ]; then
		continue
	fi
	case "${line}" in
		WORKSPACE_NAME=*|WORKSPACE_SLUG=*|APP_PORT=*|DB_PORT=*|SMTP_PORT=*|MAILPIT_PORT=*|RIVERUI_PORT=*|TEMPL_PROXY_PORT=*|COMPOSE_PROJECT_NAME=*|PG_URL=*)
			continue
			;;
	esac
	printf '%s\n' "${line}" >> "${tmp_file}"
done < "${source_env_file}"

if [ -s "${tmp_file}" ]; then
	printf '\n' >> "${tmp_file}"
fi

cat >> "${tmp_file}" <<EOF
${generated_start}
WORKSPACE_NAME=${workspace_name}
WORKSPACE_SLUG=${workspace_slug}
APP_PORT=${app_port}
DB_PORT=${db_port}
SMTP_PORT=${smtp_port}
MAILPIT_PORT=${mailpit_port}
RIVERUI_PORT=${riverui_port}
TEMPL_PROXY_PORT=${templ_proxy_port}
COMPOSE_PROJECT_NAME=untils_${workspace_slug}
PG_URL=postgresql://root:root@localhost:${db_port}/untils_dev
${generated_end}
EOF

mv "${tmp_file}" "${dest_env_file}"

cat > "${test_env_file}" <<EOF
${generated_start}
PG_URL=postgresql://root:root@localhost:${db_port}/untils_test
${generated_end}
EOF
