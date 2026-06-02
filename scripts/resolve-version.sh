#!/usr/bin/env bash
# Resolve which indexnow release to install.
#
# Precedence:
#   1. INPUT_VERSION env (from action input "version")
#   2. GITHUB_ACTION_REF, if it looks like vX.Y.Z
#   3. "latest" via GitHub Releases API
#
# Writes `version=vX.Y.Z` to $GITHUB_OUTPUT.
set -euo pipefail

v="${INPUT_VERSION:-}"

if [[ -z "$v" ]]; then
  ref="${GITHUB_ACTION_REF:-}"
  if [[ "$ref" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    v="$ref"
  fi
fi

if [[ -z "$v" || "$v" == "latest" ]]; then
  if ! command -v gh >/dev/null 2>&1; then
    echo "::error::gh CLI not available on runner; cannot resolve 'latest' version" >&2
    exit 1
  fi
  v=$(GH_TOKEN="${GH_TOKEN:-${GITHUB_TOKEN:-}}" gh api repos/jtprogru/indexnow/releases/latest --jq .tag_name)
fi

# Normalize: ensure leading v.
[[ "$v" == v* ]] || v="v${v}"

echo "version=${v}" >> "$GITHUB_OUTPUT"
echo "Resolved indexnow version: ${v}"
