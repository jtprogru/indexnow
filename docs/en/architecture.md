# Architecture

```
indexnow/
├── cmd/indexnow/main.go     # cobra wiring, ENV defaults, exit-code plumbing
└── internal/
    ├── cli/                 # logic in a testable form (no cobra deps)
    │   ├── submit.go        # RunSubmit, URL collection, output rendering, fail-on
    │   └── errors.go
    └── client/              # IndexNow HTTP client
        ├── client.go        # Submit / SubmitBatch, retry loop, payload marshalling
        ├── endpoints.go     # alias → URL resolution
        ├── retry.go         # backoff / jitter math
        └── errors.go
```

## Separation of concerns

- **`cmd/indexnow`** owns cobra and OS plumbing only: flag definitions, signal handling, ENV fallbacks, mapping `cli.Exit*` to process exit codes.
- **`internal/cli`** owns the CLI behavior in a form that is testable without spawning a binary: `RunSubmit(ctx, opts, stdin, stdout, stderr, factory) int`. Inputs are read from `io.Reader`, output from `io.Writer`, the HTTP client is provided via `SubmitterFactory` — so tests don't need a network.
- **`internal/client`** owns the IndexNow wire format and retry policy. Concurrency-safe; configured via `client.Config`.

## Retry policy

The client retries on HTTP 429, 5xx, and transport errors. Backoff is exponential with jitter, bounded by `BaseBackoff` and `MaxBackoff`. `Retry-After` is parsed as both seconds and HTTP-date. Each `Result` carries the final `StatusCode`, `Attempts`, the URLs in the batch, and the terminal error (if any).

## Batching

`SubmitBatch` splits its input at `MaxBatchSize = 10000` (protocol limit) and emits one `Result` per HTTP call, in submission order.
