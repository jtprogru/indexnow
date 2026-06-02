#!/usr/bin/env bash
# Execute `indexnow submit` with the action's inputs, parse JSON output,
# emit summary + outputs, propagate exit code.
set -euo pipefail

args=(submit --output json)

# URL source.
url_tmp=""
if [[ -n "${INPUT_URLS:-}" ]]; then
  url_tmp=$(mktemp)
  # printf preserves embedded blank lines; CLI ignores blanks and # comments.
  printf '%s\n' "$INPUT_URLS" > "$url_tmp"
  args+=(--file "$url_tmp")
elif [[ -n "${INPUT_FILE:-}" ]]; then
  args+=(--file "$INPUT_FILE")
elif [[ -n "${INPUT_SITEMAP:-}" ]]; then
  args+=(--sitemap "$INPUT_SITEMAP")
elif [[ -n "${INPUT_URLS_FROM:-}" ]]; then
  url_tmp=$(mktemp)
  # User-supplied bash snippet. Its stderr passes through to the step log;
  # we only capture stdout. set -e propagates a non-zero exit from the snippet.
  bash -c "$INPUT_URLS_FROM" > "$url_tmp"

  # Empty output is a legitimate "nothing to do" case; skip submit entirely.
  # grep -c returns 1 on zero matches under set -e, so OR with true.
  effective=$(grep -cvE '^[[:space:]]*(#|$)' "$url_tmp" || true)
  if [[ "${effective:-0}" == "0" ]]; then
    rm -f "$url_tmp"
    echo "::notice::urls-from produced no URLs; nothing to submit"
    {
      echo "## indexnow"
      echo ""
      echo "- urls-from: empty input, submit skipped"
      echo "- version: \`${VERSION:-?}\`"
    } >> "${GITHUB_STEP_SUMMARY:-/dev/null}"
    {
      echo "exit-code=0"
      echo "submitted-count=0"
      echo "failed-count=0"
      echo "report=indexnow ${VERSION:-?}: urls-from empty, submit skipped"
    } >> "${GITHUB_OUTPUT:-/dev/null}"
    exit 0
  fi

  args+=(--file "$url_tmp")
fi

[[ -n "${INPUT_CONFIG:-}"          ]] && args+=(--config          "$INPUT_CONFIG")
[[ -n "${INPUT_SITEMAP_SINCE:-}"   ]] && args+=(--sitemap-since   "$INPUT_SITEMAP_SINCE")
[[ -n "${INPUT_SITEMAP_TIMEOUT:-}" ]] && args+=(--sitemap-timeout "$INPUT_SITEMAP_TIMEOUT")
[[ -n "${INPUT_FAIL_ON:-}"         ]] && args+=(--fail-on         "$INPUT_FAIL_ON")
[[ -n "${INPUT_MAX_RETRIES:-}"     ]] && args+=(--max-retries     "$INPUT_MAX_RETRIES")
[[ -n "${INPUT_BASE_BACKOFF:-}"    ]] && args+=(--base-backoff    "$INPUT_BASE_BACKOFF")
[[ -n "${INPUT_MAX_BACKOFF:-}"     ]] && args+=(--max-backoff     "$INPUT_MAX_BACKOFF")

[[ "${INPUT_VERBOSE:-false}" == "true" ]] && args+=(--verbose)
[[ "${INPUT_DRY_RUN:-false}" == "true" ]] && args+=(--dry-run)

# Run. INDEXNOW_KEY / INDEXNOW_HOST / INDEXNOW_KEY_LOCATION / INDEXNOW_ENDPOINT
# / INDEXNOW_USER_AGENT are passed via env from action.yml, so we don't repeat
# them as flags here (avoids exposing the key to `set -x` debugging).
# stderr passes straight through to the workflow's step log; we only capture stdout.
set +e
out=$(indexnow "${args[@]}")
code=$?
set -e

[[ -n "$url_tmp" && -f "$url_tmp" ]] && rm -f "$url_tmp"

# Parse counts. With --dry-run the CLI prints plain text, not JSON — guard.
submitted=0
failed=0
if [[ "${INPUT_DRY_RUN:-false}" != "true" ]] && command -v jq >/dev/null 2>&1; then
  if echo "$out" | jq -e . >/dev/null 2>&1; then
    submitted=$(echo "$out" | jq '[.[].urlCount // 0] | add // 0')
    failed=$(echo "$out"   | jq '[.[] | select((.status // 0) < 200 or (.status // 0) >= 300 or (.error // "") != "")] | length')
  fi
fi

# Step summary.
{
  echo "## indexnow"
  echo ""
  echo "| Field | Value |"
  echo "|---|---|"
  echo "| version | \`${VERSION:-?}\` |"
  echo "| submitted | ${submitted} |"
  echo "| failed | ${failed} |"
  echo "| exit code | ${code} |"
  if [[ "${INPUT_DRY_RUN:-false}" == "true" ]]; then
    echo "| mode | dry-run |"
  fi
} >> "${GITHUB_STEP_SUMMARY:-/dev/null}"

# Outputs.
report="indexnow ${VERSION:-?}: submitted=${submitted} failed=${failed} exit=${code}"
{
  echo "exit-code=${code}"
  echo "submitted-count=${submitted}"
  echo "failed-count=${failed}"
  echo "report=${report}"
} >> "${GITHUB_OUTPUT:-/dev/null}"

# Echo CLI output to stdout unless quiet.
if [[ "${INPUT_QUIET:-false}" != "true" ]]; then
  echo "$out"
fi

exit "$code"
