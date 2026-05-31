# indexnow

**indexnow** is a tiny CLI that pushes URL-change notifications to the [IndexNow](https://www.indexnow.org/documentation) protocol — one HTTP call, shared across every participating search engine (Bing, Yandex, Naver, Seznam, Yep, ...).

It is built for content pipelines: deterministic, scriptable, fail-loud, no hidden state.

---

## What you get

- **One subcommand, three input modes** — positional args, `--file`, or `--stdin`.
- **Six endpoints out of the box** — `api`, `bing`, `yandex`, `naver`, `seznam`, `yep` — or pass a full URL.
- **Retries that respect the protocol** — exponential backoff with jitter, `Retry-After` (seconds + HTTP-date) on 429 / 5xx / transport errors.
- **Batching at the spec limit** — automatic splitting at 10 000 URLs per POST.
- **Pipeline-friendly output** — `--output text|json`, exit policy via `--fail-on any|4xx|5xx|never`.
- **ENV fallbacks** — `INDEXNOW_KEY`, `INDEXNOW_HOST`, `INDEXNOW_KEY_LOCATION`, `INDEXNOW_ENDPOINT`.

## At a glance

```bash
# Single URL, minimal flags
export INDEXNOW_KEY=8f7e6d5c4b3a29180706050403020100
indexnow submit https://example.com/posts/new

# Bulk from file, JSON output for jq
indexnow submit --file urls.txt --endpoint bing --output json | jq '.[].status'

# Pipe from a sitemap walker
sitemap-to-urls | indexnow submit --stdin --fail-on 5xx
```

## Next steps

- **[Getting started](getting-started.md)** — install + first call.
- **[Commands](commands/index.md)** — full reference for `indexnow submit`.
- **[Guides](guides/endpoints.md)** — endpoints, configuration, ENV.
