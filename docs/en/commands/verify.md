# `indexnow verify`

Verify that the hosted IndexNow key file matches the expected key. Useful before the first submission, or as a smoke test in a deploy pipeline after rotating the key file.

## Synopsis

```bash
indexnow verify [flags]
```

The command performs a single HTTP `GET` against the hosted key URL and compares the trimmed response body to `--key`. No retries — this is a one-shot diagnostic.

## URL resolution

- If `--key-location` is provided, that absolute URL is fetched.
- Otherwise, the conventional location `https://<host>/<key>.txt` is derived from `--host` and `--key`. Both are required in that case, and `--key` must satisfy IndexNow's key format (`[A-Za-z0-9-]{8,128}`).

## Flags

| Flag                                | Purpose                                                                 |
|-------------------------------------|-------------------------------------------------------------------------|
| `--key`                             | IndexNow key (env: `INDEXNOW_KEY`)                                      |
| `--host`                            | Site host, e.g. `example.com` (env: `INDEXNOW_HOST`)                    |
| `--key-location`                    | Absolute URL to the hosted key file (env: `INDEXNOW_KEY_LOCATION`)      |
| `--config PATH`                     | Yaml config with `host`/`key`/`key_location`/`user_agent` defaults      |
| `--user-agent STRING`               | HTTP `User-Agent` header (env: `INDEXNOW_USER_AGENT`; default: `indexnow/<version>`) |
| `--timeout DURATION`                | HTTP timeout for the key fetch (default `10s`)                          |
| `--output text\|json`               | Output format                                                            |
| `-q, --quiet`                       | Suppress stdout; rely on exit code                                      |
| `-v, --verbose`                     | Log lifecycle events to stderr (slog text format)                       |

## Examples

```bash
# Implicit URL: https://example.com/<KEY>.txt
indexnow verify --host example.com --key abcdef1234567890

# Custom hosting path
indexnow verify --key abcdef1234567890 \
  --key-location https://static.example.com/indexnow/key.txt

# Scripted: silent on success, exit code is the truth
indexnow verify -q --host example.com --key abcdef1234567890 \
  && echo "key live" || echo "fix the key file"
```

## Output

Text mode prints a single line:

```
OK:   https://example.com/abcdef1234567890.txt
FAIL: https://example.com/abcdef1234567890.txt status=404 err=http 404
FAIL: https://example.com/abcdef1234567890.txt status=200 err=hosted key does not match expected
```

JSON mode emits a single object: `{ "url", "ok", "status", "hosted", "error" }`. The `hosted` field is populated on mismatch (truncated to 80 chars) so you can eyeball what's actually being served.

## Exit codes

- `0` — hosted key matches
- `1` — mismatch, non-200, or network error
- `2` — usage error (missing key, malformed `--key-location`, etc.)
