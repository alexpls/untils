#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

if [[ $# -gt 0 ]]; then
  tag="$1"
else
  tag="$(gh api repos/starfederation/datastar/releases/latest --jq .tag_name)"
fi

version="${tag#v}"
url="https://cdn.jsdelivr.net/gh/starfederation/datastar@${version}/bundles/datastar.js"
out_file="assets/js/vendor/datastar.js"
tmp_file="$(mktemp)"

extract_version() {
  sed -n '1s|^// Datastar v\(.*\)$|\1|p' "$1"
}

current_version=""
if [[ -f "$out_file" ]]; then
  current_version="$(extract_version "$out_file")"
fi

if [[ -n "$current_version" && "$current_version" == "$version" ]]; then
  rm -f "$tmp_file"
  echo "datastar already up to date (v${current_version})"
  exit 0
fi

curl -fsSL "$url" -o "$tmp_file"

if ! rg -q "Datastar v" "$tmp_file"; then
  echo "downloaded file does not look like a datastar bundle: $url" >&2
  rm -f "$tmp_file"
  exit 1
fi

new_version="$(extract_version "$tmp_file")"
if [[ -n "$current_version" && -n "$new_version" && "$current_version" == "$new_version" ]]; then
  rm -f "$tmp_file"
  echo "datastar already up to date (v${current_version})"
  exit 0
fi

mv "$tmp_file" "$out_file"

echo "updated $out_file from $url"
head -n 1 "$out_file"
