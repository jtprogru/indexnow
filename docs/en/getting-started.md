# Getting started

## Install

Homebrew (macOS / Linux):

```bash
brew tap jtprogru/tap
brew install indexnow
```

`go install`:

```bash
go install github.com/jtprogru/indexnow/cmd/indexnow@latest
```

Or grab a prebuilt binary from [Releases](https://github.com/jtprogru/indexnow/releases).

## Get an IndexNow key

The protocol requires a key (8..128 chars, `[A-Za-z0-9-]`) hosted at a known URL on your site — typically `https://example.com/<key>.txt` containing the key as the only line. See the [IndexNow docs](https://www.indexnow.org/documentation) for details.

```bash
export INDEXNOW_KEY=8f7e6d5c4b3a29180706050403020100
```

## First call

```bash
indexnow submit https://example.com/posts/new
```

That's it. The host is inferred from the URL; the default endpoint is `https://api.indexnow.org/indexnow`, which shares your submission with every participating search engine per the protocol spec.

## Dry-run

```bash
indexnow submit --dry-run --file urls.txt
```

Prints what would be sent and exits with `0` — no network call.
