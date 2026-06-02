# GitHub Action

Use indexnow as a step in any GitHub Actions workflow. The action downloads the matching release binary for the runner's OS/arch, verifies its sha256 against `checksums.txt` from the same release, and runs `indexnow submit` with the inputs you pass.

```yaml
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    sitemap: https://example.com/sitemap.xml
```

Pin to a major (`@v0`) for floating-but-stable, a tag (`@v0.5.0`) for exactness, or a commit SHA for supply-chain paranoia.

## Supported runners

| `runs-on` | Status |
|---|---|
| `ubuntu-latest`, `ubuntu-*` (x64, arm64) | supported |
| `macos-latest`, `macos-*` (x64, arm64) | supported |
| `windows-*` | unsupported — preflight fails fast with a pointer to ubuntu/macos |

## Inputs

| Name | Required | Default | Notes |
|---|---|---|---|
| `key` | yes | — | IndexNow key (8..128, `[A-Za-z0-9-]`). Passed via env, not flags. |
| `urls` | one of | — | Newline-separated URLs. |
| `file` | one of | — | Path to a file with one URL per line. |
| `sitemap` | one of | — | URL or local path to `sitemap.xml` (`.gz` and `<sitemapindex>` followed). |
| `sitemap-since` | no | — | RFC3339; drop entries with older `<lastmod>`. |
| `sitemap-timeout` | no | CLI default (`30s`) | Per-request HTTP timeout for sitemap fetches. |
| `host` | no | inferred from first URL | Site host (e.g. `example.com`). |
| `key-location` | no | derived from `host` + `key` | Absolute URL to the hosted key file. |
| `endpoint` | no | `api` | Alias or full URL; comma-separated for fan-out. |
| `user-agent` | no | `indexnow/<version>` | Sent as `User-Agent`. |
| `fail-on` | no | `any` | `any\|4xx\|5xx\|never` — which response classes set exit ≠ 0. |
| `quiet` | no | `false` | Suppress the CLI's stdout in the step log. |
| `verbose` | no | `false` | Emit slog lifecycle/retry events to stderr. |
| `dry-run` | no | `false` | Print what would be sent; do not call the endpoint. |
| `max-retries` | no | CLI default | Retries on 429/5xx/transport. |
| `base-backoff` | no | CLI default | Initial backoff (Go duration, e.g. `200ms`). |
| `max-backoff` | no | CLI default | Cap on backoff. |
| `version` | no | action's tag | indexnow release to install. `"latest"` resolves at run time. |
| `github-token` | no | `${{ github.token }}` | Only used to call the GitHub Releases API. |

Exactly one of `urls` / `file` / `sitemap` must be set — preflight fails otherwise.

## Outputs

| Name | Notes |
|---|---|
| `exit-code` | Exit code of `indexnow submit`. |
| `submitted-count` | Sum of `urlCount` across all batches. `0` in `dry-run`. |
| `failed-count` | Batches that finished non-2xx or with an error. `0` in `dry-run`. |
| `report` | One-line summary; also written to `$GITHUB_STEP_SUMMARY`. |

## Recipes

### On every push to `content/`

```yaml
name: indexnow
on:
  push:
    branches: [main]
    paths: ["content/**"]

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - uses: jtprogru/indexnow@v0
        with:
          key: ${{ secrets.INDEXNOW_KEY }}
          sitemap: https://example.com/sitemap.xml
          sitemap-since: ${{ github.event.before }}
          endpoint: bing,yandex
```

`sitemap-since: ${{ github.event.before }}` is a string; the action passes it through to `--sitemap-since`, which parses RFC3339. For pushes that's the commit timestamp of the previous head — entries with older `<lastmod>` are skipped, the rest go out.

### Hourly schedule

```yaml
on:
  schedule:
    - cron: "0 * * * *"

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - uses: jtprogru/indexnow@v0
        with:
          key: ${{ secrets.INDEXNOW_KEY }}
          sitemap: https://example.com/sitemap.xml
          sitemap-since: ${{ format('{0}-{1}-{2}T{3}:00:00Z', …) }}
```

If you don't want to fiddle with date arithmetic in YAML, drop `sitemap-since` — every hourly run re-submits the whole sitemap. IndexNow is idempotent; the cost is the HTTP call.

### After Hugo / Eleventy build, explicit URL list

```yaml
- name: Build
  run: hugo --minify

- name: Collect changed URLs
  id: changed
  run: |
    {
      echo 'urls<<EOF'
      git diff --name-only "${{ github.event.before }}" HEAD -- 'content/**/*.md' \
        | sed 's#^content/\(.*\)\.md$#https://example.com/\1/#'
      echo EOF
    } >> "$GITHUB_OUTPUT"

- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    host: example.com
    urls: ${{ steps.changed.outputs.urls }}
```

The path-to-URL mapping is project-specific (Hugo's `permalinks`, Eleventy's `permalink` front-matter, …). A built-in `from-diff` subcommand is on the roadmap.

## Secrets

Put `INDEXNOW_KEY` in repository or org-level secrets. The action takes it as an `input` and passes it through env (`INDEXNOW_KEY`) — the value never appears in command-line traces, even with `set -x`.

Hosting the key file: drop a `<key>.txt` next to your static site's root that contains exactly the key. The `verify` subcommand on the CLI checks this; it isn't yet exposed as an action input — call `indexnow verify` as a follow-up step if you need the assertion in CI.

## How install works

On the first use in a job:

1. Resolve version (`inputs.version` → action's tag → `latest`).
2. Download `indexnow_{Linux|Darwin}_{x86_64|arm64}.tar.gz` and `checksums.txt` from the release.
3. Verify sha256 (`sha256sum -c` on Linux, `shasum -a 256` on macOS).
4. Extract to `$RUNNER_TOOL_CACHE/indexnow/<version>/<os-arch>/`.
5. Cache that directory via `actions/cache` keyed by version+os+arch — subsequent jobs (and reruns) skip the download.

A repository pinning the action by commit SHA inherits the GPG-signed `checksums.txt.sig` only when verifying out-of-band; the action itself relies on TLS to GitHub Releases plus sha256.
