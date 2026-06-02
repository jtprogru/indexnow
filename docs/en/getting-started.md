# Getting started

Two ways to run indexnow. Pick whichever fits your pipeline — they share the same engine and the same protocol semantics.

## Get an IndexNow key

Both paths need a key (8..128 chars, `[A-Za-z0-9-]`) hosted at a known URL on your site — typically `https://example.com/<key>.txt` containing the key as the only line. See the [IndexNow docs](https://www.indexnow.org/documentation) for details.

A quick way to generate one:

```bash
openssl rand -hex 32 > public/8f7e6d5c4b3a29180706050403020100.txt
echo "8f7e6d5c4b3a29180706050403020100" > public/8f7e6d5c4b3a29180706050403020100.txt
```

(File name and contents must match.) Deploy the file with your site, then store the key value as a CI secret — usually `INDEXNOW_KEY`.

## Path A — GitHub Action (recommended)

If your content lives in a GitHub repo, the action is the shortest path:

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
```

That's it. `sitemap-since: ${{ github.event.before }}` means "only URLs whose `<lastmod>` is newer than the previous HEAD" — entries older than that are skipped.

→ See **[GitHub Action](guides/github-action.md)** for all inputs/outputs and more recipes (scheduled runs, after-Hugo/Eleventy builds, explicit URL lists).

## Path B — CLI

Use the CLI when you run indexnow locally, from a cron job, or from a CI that isn't GitHub Actions.

### Install

Homebrew (macOS / Linux):

```bash
brew tap jtprogru/tap
brew install indexnow
```

`go install`:

```bash
go install github.com/jtprogru/indexnow/cmd/indexnow@latest
```

Or grab a prebuilt binary from [Releases](https://github.com/jtprogru/indexnow/releases) (Linux / macOS / FreeBSD, amd64 / arm64).

### First call

```bash
export INDEXNOW_KEY=8f7e6d5c4b3a29180706050403020100
indexnow submit https://example.com/posts/new
```

Host is inferred from the URL; the default endpoint is `https://api.indexnow.org/indexnow`, which shares your submission with every participating search engine per the protocol spec.

### Dry-run

```bash
indexnow submit --dry-run --file urls.txt
```

Prints what would be sent and exits with `0` — no network call. Handy for validating a sitemap pull before flipping the switch in production.

→ Full command reference: **[CLI commands](commands/index.md)**.
