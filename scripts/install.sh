#!/usr/bin/env bash
# Download and install the indexnow binary from a GitHub release.
#
# Inputs (env):
#   VERSION              — vX.Y.Z tag (required)
#   OS                   — Linux | Darwin (required)
#   ARCH                 — x86_64 | arm64 (required)
#   RUNNER_TOOL_CACHE    — provided by Actions runner
#
# Verifies sha256 against checksums.txt published alongside the release.
set -euo pipefail

: "${VERSION:?VERSION is required}"
: "${OS:?OS is required}"
: "${ARCH:?ARCH is required}"
: "${RUNNER_TOOL_CACHE:?RUNNER_TOOL_CACHE is required}"

base="https://github.com/jtprogru/indexnow/releases/download/${VERSION}"
archive="indexnow_${OS}_${ARCH}.tar.gz"
sums="checksums.txt"

dest="${RUNNER_TOOL_CACHE}/indexnow/${VERSION}/${OS}-${ARCH}"
mkdir -p "$dest"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${base}/${archive}"
curl -fsSL --retry 3 --retry-delay 2 -o "${tmp}/${archive}" "${base}/${archive}"

echo "Downloading ${base}/${sums}"
curl -fsSL --retry 3 --retry-delay 2 -o "${tmp}/${sums}" "${base}/${sums}"

# sha256sum is GNU coreutils on Linux; on macOS we fall back to shasum -a 256.
if command -v sha256sum >/dev/null 2>&1; then
  ( cd "$tmp" && grep " ${archive}\$" "${sums}" | sha256sum -c - )
else
  expected=$(grep " ${archive}\$" "${tmp}/${sums}" | awk '{print $1}')
  actual=$(shasum -a 256 "${tmp}/${archive}" | awk '{print $1}')
  if [[ "$expected" != "$actual" ]]; then
    echo "::error::sha256 mismatch for ${archive}: expected ${expected}, got ${actual}" >&2
    exit 1
  fi
fi
echo "Checksum verified for ${archive}"

tar -xzf "${tmp}/${archive}" -C "$dest"
chmod +x "${dest}/indexnow"

# Sanity check.
# `indexnow --version` prints `indexnow X.Y.Z (commit … built … by goreleaser)`.
reported=$("${dest}/indexnow" --version 2>&1 | awk '{print $2}' || true)
case "$reported" in
  "$VERSION"|"${VERSION#v}")
    echo "Installed indexnow ${reported} → ${dest}/indexnow"
    ;;
  *)
    echo "::warning::installed binary reports version='${reported}', expected '${VERSION}'"
    ;;
esac
