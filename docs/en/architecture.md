# Architecture

```
indexnow/
├── action.yml               # composite GitHub Action definition (inputs/outputs/steps)
├── scripts/                 # the action's step scripts
│   ├── resolve-version.sh   # input → action ref → "latest"
│   ├── detect-platform.sh   # RUNNER_OS/ARCH → goreleaser archive naming
│   ├── install.sh           # download + sha256-verify + extract into the tool cache
│   ├── preflight.sh         # exactly-one-URL-source check
│   └── run.sh               # build the submit argv, parse JSON, emit outputs/summary
├── cmd/indexnow/main.go     # cobra wiring, ENV/config defaults, exit-code plumbing
└── internal/
    ├── cli/                 # logic in a testable form (no cobra deps)
    │   ├── submit.go        # RunSubmit, URL collection, endpoint fan-out, rendering, fail-on
    │   ├── keygen.go        # RunKeygen, hosted key file writing
    │   ├── verify.go        # RunVerify, hosted key check
    │   └── errors.go
    ├── client/              # IndexNow HTTP client
    │   ├── client.go        # Submit / SubmitBatch, retry loop, payload marshalling
    │   ├── endpoints.go     # alias → URL resolution (single and comma-separated)
    │   ├── retry.go         # backoff / jitter math
    │   └── errors.go
    ├── config/config.go     # optional yaml config (XDG default path or --config)
    └── sitemap/sitemap.go   # streaming urlset / sitemapindex parser
```

## Separation of concerns

- **`cmd/indexnow`** owns cobra and OS plumbing only: flag definitions, signal handling, the flag > env > config > default merge, mapping `cli.Exit*` to process exit codes.
- **`internal/cli`** owns the CLI behavior in a form that is testable without spawning a binary: `RunSubmit(ctx, opts, stdin, stdout, stderr, factory) int`, plus `RunKeygen` and `RunVerify` in the same shape. Inputs are read from `io.Reader`, output from `io.Writer`, the HTTP client is provided via `SubmitterFactory` — so tests don't need a network.
- **`internal/client`** owns the IndexNow wire format and retry policy. Concurrency-safe; configured via `client.Config`.
- **`internal/config`** owns the yaml file: strict decoding (unknown fields rejected), the `$XDG_CONFIG_HOME/indexnow/config.yaml` default path, and the "missing default file is not an error, missing explicit `--config` is" distinction.
- **`internal/sitemap`** owns the sitemap.org wire format. It matches element local names, so documents without an `xmlns` parse too, and streams through `encoding/xml` so a 50 MB / 50 000-entry sitemap never loads whole into memory. `<sitemapindex>` recursion is capped at `MaxDepth = 5` and visited sources are deduped, so self-referential indexes terminate.
- **`action.yml` + `scripts/`** wrap the released binary rather than re-implementing anything: the action resolves a version, verifies a checksum, and shells out to `indexnow submit --output json`.

## Endpoint fan-out

`--endpoint` resolves to a list; each entry gets its own goroutine and its own `client.Client`, all sharing the same URL batch. Results come back as one `endpointBatch` per endpoint, in input order, so output stays deterministic regardless of which endpoint answers first. Wall-time tracks the slowest endpoint, not the sum.

## Retry policy

The client retries on HTTP 429, 5xx, and transport errors. Backoff is exponential with jitter, bounded by `BaseBackoff` and `MaxBackoff`. `Retry-After` is parsed as both seconds and HTTP-date. Each `Result` carries the final `StatusCode`, `Attempts`, the URLs in the batch, and the terminal error (if any).

## Batching

`SubmitBatch` splits its input at `MaxBatchSize = 10000` (protocol limit) and emits one `Result` per HTTP call, in submission order.
