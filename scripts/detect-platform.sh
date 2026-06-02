#!/usr/bin/env bash
# Map GHA RUNNER_OS / RUNNER_ARCH to the goreleaser archive naming.
#
# goreleaser produces:
#   indexnow_Linux_x86_64.tar.gz
#   indexnow_Linux_arm64.tar.gz
#   indexnow_Darwin_x86_64.tar.gz
#   indexnow_Darwin_arm64.tar.gz
#
# Writes `os` (Linux|Darwin) and `arch` (x86_64|arm64) to $GITHUB_OUTPUT.
set -euo pipefail

case "${RUNNER_OS:-}" in
  Linux)  os=Linux ;;
  macOS)  os=Darwin ;;
  Windows)
    echo "::error::indexnow Action does not support Windows runners. Use ubuntu-latest or macos-latest." >&2
    exit 1
    ;;
  *)
    echo "::error::Unsupported RUNNER_OS='${RUNNER_OS:-}'" >&2
    exit 1
    ;;
esac

case "${RUNNER_ARCH:-}" in
  X64)   arch=x86_64 ;;
  ARM64) arch=arm64 ;;
  *)
    echo "::error::Unsupported RUNNER_ARCH='${RUNNER_ARCH:-}' (need X64 or ARM64)" >&2
    exit 1
    ;;
esac

{
  echo "os=${os}"
  echo "arch=${arch}"
} >> "$GITHUB_OUTPUT"
echo "Resolved platform: ${os}/${arch}"
