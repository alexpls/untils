#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "" ]; then
	echo "usage: $0 <workspace-name>" >&2
	exit 1
fi

workspace_name="$1"
if [ "${workspace_name}" = "default" ]; then
	echo "refusing to delete default workspace" >&2
	exit 1
fi

if ! command -v jj >/dev/null 2>&1; then
	echo "jj is required" >&2
	exit 1
fi

repo_root="$(jj root)"

if ! jj workspace list -T 'name ++ "\n"' | grep -Fxq "${workspace_name}"; then
	echo "workspace '${workspace_name}' not found" >&2
	exit 1
fi

workspace_root=""
if workspace_root="$(jj workspace root --name "${workspace_name}" 2>/dev/null)"; then
	:
else
	repo_parent="$(dirname "${repo_root}")"
	repo_name="$(basename "${repo_root}")"
	workspace_root="${repo_parent}/${repo_name}-${workspace_name}"
fi

echo "workspace: ${workspace_name}"
echo "workspace root: ${workspace_root}"

if [ -d "${workspace_root}" ] && [ -f "${workspace_root}/mise.toml" ]; then
	(
		cd "${workspace_root}"
		mise run dev:down || true
	)
fi

jj workspace forget "${workspace_name}"

if [ -d "${workspace_root}" ]; then
	rm -rf "${workspace_root}"
fi

workspace_slug="$(printf '%s' "${workspace_name}" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')"
if [ -z "${workspace_slug}" ]; then
	workspace_slug="workspace"
fi

docker volume rm "untils_${workspace_slug}_pgdata" >/dev/null 2>&1 || true

echo "workspace '${workspace_name}' deleted"
