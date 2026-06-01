# `indexnow submit`

Submit one or more URLs to an IndexNow endpoint.

## Synopsis

```bash
indexnow submit [urls...] [flags]
```

Exactly one URL source is required:

- positional args: `indexnow submit URL [URL ...]`
- `--file PATH` — one URL per line, `#` for comments, blank lines skipped
- `--stdin` — read URLs from stdin
- `--sitemap URL|PATH` — fetch URLs from a sitemap (sitemapindex is followed recursively, `.gz` is gunzipped). Optional `--sitemap-since <RFC3339>` filters entries by `<lastmod>`; entries without `<lastmod>` always pass — absent signal is treated as "may have changed", which is the safe default for a notification protocol.

## Flags

| Flag                                | Purpose                                                                 |
|-------------------------------------|-------------------------------------------------------------------------|
| `--key`                             | IndexNow key (env: `INDEXNOW_KEY`)                                      |
| `--host`                            | Site host, e.g. `example.com` (env: `INDEXNOW_HOST`; inferred otherwise) |
| `--key-location`                    | Absolute URL to the hosted key file (env: `INDEXNOW_KEY_LOCATION`)      |
| `--endpoint`                        | One alias or full URL, or a comma-separated list (parallel fan-out)     |
| `--config PATH`                     | Yaml config with `host`/`key`/`key_location`/`endpoint`/`user_agent` defaults |
| `--user-agent STRING`               | HTTP `User-Agent` header (env: `INDEXNOW_USER_AGENT`; default: `indexnow/<version>`) |
| `--file PATH`                       | Read URLs from file                                                     |
| `--stdin`                           | Read URLs from stdin                                                    |
| `--sitemap URL\|PATH`               | Fetch URLs from a sitemap (sitemapindex recursed; `.gz` gunzipped)      |
| `--sitemap-since RFC3339`           | Filter sitemap entries by `<lastmod>` (entries without lastmod always pass) |
| `--sitemap-timeout DURATION`        | Per-request HTTP timeout for sitemap fetches (default `30s`)            |
| `--dry-run`                         | Print what would be sent and exit                                       |
| `--output text\|json`               | Output format                                                            |
| `-q, --quiet`                       | Suppress stdout; rely on exit code (errors still go to stderr)          |
| `-v, --verbose`                     | Log submit lifecycle and retry events to stderr (slog text format)      |
| `--fail-on any\|4xx\|5xx\|never`    | Which responses trigger exit 1                                          |
| `--max-retries N`                   | Retries on 429 / 5xx / transport errors (default `3`)                   |
| `--base-backoff DURATION`           | Base retry backoff (default `1s`)                                       |
| `--max-backoff DURATION`            | Max retry backoff (default `30s`)                                       |

## Examples

```bash
indexnow submit https://example.com/post/1
indexnow submit --file urls.txt --endpoint bing
cat urls.txt | indexnow submit --stdin --output json
indexnow submit --endpoint bing,yandex https://example.com/post/1
indexnow submit -q https://example.com/post/1 && echo ok || echo failed
indexnow submit --sitemap https://example.com/sitemap.xml
indexnow submit --sitemap sitemap.xml.gz --sitemap-since 2026-05-01T00:00:00Z
```

## Exit codes

- `0` — OK
- `1` — submission failed (HTTP / network / fail-on triggered)
- `2` — usage error
