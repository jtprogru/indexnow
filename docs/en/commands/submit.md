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

## Flags

| Flag                                | Purpose                                                                 |
|-------------------------------------|-------------------------------------------------------------------------|
| `--key`                             | IndexNow key (env: `INDEXNOW_KEY`)                                      |
| `--host`                            | Site host, e.g. `example.com` (env: `INDEXNOW_HOST`; inferred otherwise) |
| `--key-location`                    | Absolute URL to the hosted key file (env: `INDEXNOW_KEY_LOCATION`)      |
| `--endpoint`                        | `api`, `bing`, `yandex`, `naver`, `seznam`, `yep`, or a full URL        |
| `--file PATH`                       | Read URLs from file                                                     |
| `--stdin`                           | Read URLs from stdin                                                    |
| `--dry-run`                         | Print what would be sent and exit                                       |
| `--output text\|json`               | Output format                                                            |
| `--fail-on any\|4xx\|5xx\|never`    | Which responses trigger exit 1                                          |
| `--max-retries N`                   | Retries on 429 / 5xx / transport errors (default `3`)                   |
| `--base-backoff DURATION`           | Base retry backoff (default `1s`)                                       |
| `--max-backoff DURATION`            | Max retry backoff (default `30s`)                                       |

## Examples

```bash
indexnow submit https://example.com/post/1
indexnow submit --file urls.txt --endpoint bing
cat urls.txt | indexnow submit --stdin --output json
```

## Exit codes

- `0` — OK
- `1` — submission failed (HTTP / network / fail-on triggered)
- `2` — usage error
