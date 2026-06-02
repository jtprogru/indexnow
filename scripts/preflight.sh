#!/usr/bin/env bash
# Validate that exactly one URL source is provided.
#
# Mirrors CLI validation, but fails earlier with a clearer error than the
# wrapped CLI exit code 2.
set -euo pipefail

sources=0
[[ -n "${INPUT_URLS:-}"      ]] && sources=$((sources + 1))
[[ -n "${INPUT_FILE:-}"      ]] && sources=$((sources + 1))
[[ -n "${INPUT_SITEMAP:-}"   ]] && sources=$((sources + 1))
[[ -n "${INPUT_URLS_FROM:-}" ]] && sources=$((sources + 1))

if (( sources != 1 )); then
  echo "::error::exactly one of inputs.urls / inputs.file / inputs.sitemap / inputs.urls-from must be set (got ${sources})" >&2
  exit 2
fi
