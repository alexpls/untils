#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "" ]; then
	echo "usage: $0 <workspace-name> [destination-path]" >&2
	exit 1
fi

workspace_name="$1"
destination_path="${2:-}"

if ! command -v jj >/dev/null 2>&1; then
	echo "jj is required" >&2
	exit 1
fi

repo_root="$(jj root)"
repo_parent="$(dirname "${repo_root}")"
repo_name="$(basename "${repo_root}")"

if [ -z "${destination_path}" ]; then
	destination_path="${repo_parent}/${repo_name}-${workspace_name}"
fi

if jj workspace list -T 'name ++ "\n"' | grep -Fxq "${workspace_name}"; then
	echo "workspace '${workspace_name}' already exists" >&2
	exit 1
fi

if [ -e "${destination_path}" ]; then
	echo "destination already exists: ${destination_path}" >&2
	exit 1
fi

echo "creating workspace '${workspace_name}' at ${destination_path}"
(
	cd "${repo_root}"
	jj workspace add "${destination_path}" --name "${workspace_name}" -r @
)

cd "${destination_path}"

echo "installing workspace dependencies"
bun install
mise trust

echo "starting workspace services"
mise run dev:up

echo "building frontend assets"
bun run build-css
bun run build-js

echo "waiting for postgres to accept connections"
for _ in $(seq 1 60); do
	if zsh -lc '. ./scripts/workspace-env.sh; psql "${PG_URL%/untils_dev}" -Atc "select 1"' >/dev/null 2>&1; then
		break
	fi
	sleep 1
done
if ! zsh -lc '. ./scripts/workspace-env.sh; psql "${PG_URL%/untils_dev}" -Atc "select 1"' >/dev/null 2>&1; then
	echo "postgres did not become ready in time" >&2
	exit 1
fi

echo "initializing databases"
mise run db:reset

echo "verifying readiness"
zsh -lc '. ./scripts/workspace-env.sh; psql "$PG_URL" -Atc "select email from users;"'
mise run test:unit
mise run dev:info

echo "workspace '${workspace_name}' is ready"
