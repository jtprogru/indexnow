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
| `urls-from` | one of | — | Bash snippet; its stdout is treated as the URL list. See [Custom URL sources](#custom-url-sources-via-urls-from). |
| `sitemap-since` | no | — | RFC3339; drop entries with older `<lastmod>`. |
| `sitemap-timeout` | no | CLI default (`30s`) | Per-request HTTP timeout for sitemap fetches. |
| `host` | no | inferred from first URL | Site host (e.g. `example.com`). |
| `key-location` | no | derived from `host` + `key` | Absolute URL to the hosted key file. |
| `endpoint` | no | `api` | Alias or full URL; comma-separated for fan-out. |
| `user-agent` | no | `indexnow/<version>` | Sent as `User-Agent`. |
| `config` | no | — | Path (relative to `$GITHUB_WORKSPACE`) to an indexnow yaml config — `host`, `key`, `endpoint`, etc. defaults. |
| `fail-on` | no | `any` | `any\|4xx\|5xx\|never` — which response classes set exit ≠ 0. |
| `quiet` | no | `false` | Suppress the CLI's stdout in the step log. |
| `verbose` | no | `false` | Emit slog lifecycle/retry events to stderr. |
| `dry-run` | no | `false` | Print what would be sent; do not call the endpoint. |
| `max-retries` | no | CLI default | Retries on 429/5xx/transport. |
| `base-backoff` | no | CLI default | Initial backoff (Go duration, e.g. `200ms`). |
| `max-backoff` | no | CLI default | Cap on backoff. |
| `version` | no | action's tag | indexnow release to install. `"latest"` resolves at run time. |
| `github-token` | no | `${{ github.token }}` | Only used to call the GitHub Releases API. |

Exactly one of `urls` / `file` / `sitemap` / `urls-from` must be set — preflight fails otherwise.

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

## Custom URL sources via `urls-from`

When neither `urls`, `file`, nor `sitemap` fits — produce the URL list yourself. `urls-from` is a bash snippet whose stdout becomes the URL list (one URL per line, `#`-prefixed lines are comments, blank lines ignored). Runs in `$GITHUB_WORKSPACE`, so `git`, locally-checked-out files, and other tools are immediately available.

Empty output is success — the step exits 0 with `submitted-count=0` and `submit` is not invoked. A non-zero exit from your snippet fails the step (stderr is preserved in the step log).

### Recipe: only changed URLs from `git diff`

The original motivation. The path-to-URL mapping is project-specific (Hugo permalinks, Eleventy `permalink:` front-matter, custom routers), so it lives in your snippet — not in a config schema indexnow has to maintain.

```yaml
- uses: actions/checkout@v6
  with:
    fetch-depth: 0          # required so `git diff <base>..HEAD` resolves
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    host: example.com
    urls-from: |
      git diff --name-only --diff-filter=AMR \
        "${{ github.event.before }}..${{ github.event.after }}" -- 'content/**/*.md' |
        sed 's#^content/\(.*\)\.md$#https://example.com/\1/#'
```

Watch out for the "force-push to a new branch" edge case where `github.event.before` is `0000000…` — guard with an `if:` and fall back to `origin/main..HEAD`.

### Recipe: sitemap with a content-filter

When `--sitemap-since` is not selective enough — e.g. you want only URLs whose path matches a prefix:

```yaml
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    urls-from: |
      curl -sSL https://example.com/sitemap.xml |
        grep -oE '<loc>[^<]+</loc>' |
        sed -E 's#</?loc>##g' |
        grep '^https://example.com/blog/'
```

### Recipe: pull URLs from a CMS API

```yaml
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    urls-from: |
      curl -sSL -H "Authorization: Bearer ${{ secrets.CMS_TOKEN }}" \
        https://cms.example.com/api/recently-published |
        jq -r '.items[].url'
```

`urls-from` is action-only. From the CLI, the equivalent is a plain shell pipe into `indexnow submit --stdin`:

```bash
git diff --name-only HEAD~1..HEAD -- 'content/**/*.md' |
  sed 's#^content/\(.*\)\.md$#https://example.com/\1/#' |
  indexnow submit --stdin
```

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
