#!/usr/bin/env bash
set -euo pipefail

repo="starfederation/datastar"
vendor_dir="assets/js/vendor"
datastar_js="${vendor_dir}/datastar.js"
datastar_map="${vendor_dir}/datastar.js.map"

latest_version="$(gh release view --repo "$repo" --json tagName --jq '.tagName')"
if [[ -z "$latest_version" ]]; then
	echo "could not read latest Datastar release from $repo" >&2
	exit 1
fi

tmpdir="$(mktemp -d)"
archive="${tmpdir}/datastar.zip"
source_dir="${tmpdir}/source"

mkdir -p "$source_dir"
gh api "repos/${repo}/zipball/${latest_version}" > "$archive"
unzip -q "$archive" -d "$source_dir"

extracted_dir=("${source_dir}"/*)

cp "${extracted_dir[0]}/bundles/datastar.js" "$datastar_js"
cp "${extracted_dir[0]}/bundles/datastar.js.map" "$datastar_map"

echo "Updated Datastar to $latest_version"
