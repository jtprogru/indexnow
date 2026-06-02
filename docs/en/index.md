# indexnow

**indexnow** notifies search engines (Bing, Yandex, Naver, Seznam, Yep, ...) about URL changes via the [IndexNow](https://www.indexnow.org/documentation) protocol. One HTTP call, shared across every participant per the spec — no per-engine plumbing.

Built for content pipelines: deterministic, scriptable, fail-loud, no hidden state. Ships as a **GitHub Action** for CI workflows and a **CLI** for everything else.

---

## Recommended: use as a GitHub Action

```yaml
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    sitemap: https://example.com/sitemap.xml
```

One step in any workflow. No `setup-go`, no Docker pull — the action downloads a pinned release binary for the runner's OS/arch, sha256-verifies it, caches it, and runs `indexnow submit` with your inputs. Sensitive values go through env, never command-line flags.

→ Full inputs/outputs reference and CI recipes: **[GitHub Action](guides/github-action.md)**.

## Alternative: use as a CLI

For local runs, scheduled scripts, or other CIs:

```bash
brew install jtprogru/tap/indexnow

export INDEXNOW_KEY=8f7e6d5c4b3a29180706050403020100
indexnow submit https://example.com/posts/new

# bulk from a sitemap
indexnow submit --sitemap https://example.com/sitemap.xml --sitemap-since 2026-05-01T00:00:00Z

# fan-out to several endpoints, JSON output for downstream tooling
indexnow submit --file urls.txt --endpoint bing,yandex --output json | jq '.[].status'
```

→ Install, first call, dry-run: **[Getting started](getting-started.md)**.

## What you get

- **Four URL sources** — positional args, `--file`, `--stdin`, `--sitemap` (with `.gz` and `<sitemapindex>` support, RFC3339 `--sitemap-since` filter).
- **Six endpoints out of the box** — `api`, `bing`, `yandex`, `naver`, `seznam`, `yep` — or any full URL. Comma-separated list fans out in parallel.
- **Retries that respect the protocol** — exponential backoff with jitter, `Retry-After` (seconds + HTTP-date) on 429 / 5xx / transport errors.
- **Batching at the spec limit** — automatic splitting at 10 000 URLs per POST.
- **Pipeline-friendly output** — `--output text|json`, exit policy via `--fail-on any|4xx|5xx|never`, `--quiet` for exit-code-only signalling.
- **Observability without footguns** — `--verbose` emits `slog` lifecycle/retry events to stderr, leaving stdout clean for downstream tooling.

## Where to next

- **[GitHub Action](guides/github-action.md)** — inputs, outputs, CI recipes.
- **[Getting started](getting-started.md)** — CLI install and first call.
- **[CLI commands](commands/index.md)** — reference for `submit` and `verify`.
- **[Guides](guides/endpoints.md)** — endpoints, configuration, ENV.
