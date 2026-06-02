# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The GoReleaser pipeline auto-generates per-release notes on the GitHub Releases page from commit messages; this file is the project-level human-curated history.

## [Unreleased]

### Added

- New `indexnow key` command namespace that groups key-management operations. `indexnow key gen` generates a random hex-encoded IndexNow key (default 32 chars, 128 bits of entropy from `crypto/rand`; `--length 8..128` accepted) and optionally writes the hosted key file `<dir>/<key>.txt` with `--write <dir>` (mode `0644`, refuses to overwrite without `--force`). Output is the bare key on stdout for `KEY=$(indexnow key gen)`-style use; `--output json` emits `{"key":"…","path":"…"}`; `-q` suppresses stdout while still writing the file. `indexnow key verify` is the canonical form of the existing verify operation — same flags, same behavior. No new Action input: key management is intentionally CLI-only; the Action uses keys but does not generate or rotate them.
- New documentation page `guides/key-lifecycle.md` (EN+RU) walks through the whole flow — generate, deploy, hosting-mode choice, where the key lives at use-time, verify the bootstrap, use, manual rotate. Getting Started now defers to this guide instead of duplicating partial fragments.

### Changed

- `indexnow verify` (top-level) is kept as a backwards-compatibility alias for `indexnow key verify`. Existing scripts written against v0.3.0–v0.6.x continue to work unchanged. New code should prefer the canonical `key verify` form; the alias is documented in the command reference.

### Fixed

-

## [0.6.0] — 2026-06-02

### Added

- New Action input `urls-from`: a bash snippet whose stdout is treated as the URL list (one URL per line, `#`-prefixed lines are comments, blank lines ignored). Runs in `$GITHUB_WORKSPACE`, so `git diff`, locally-checked-out files, and other tooling are immediately available. Mutually exclusive with `urls` / `file` / `sitemap`. Empty output is an explicit success — the step exits 0 with `submitted-count=0` and `submit` is not invoked, which lets CI runs on pushes that don't touch content finish cleanly. A non-zero exit from the snippet fails the step (stderr passes through to the step log). Designed for `git diff | sed` content-pipeline recipes where the path-to-URL mapping is too project-specific to bake into indexnow.
- New Action input `config`: path (relative to `$GITHUB_WORKSPACE`) to an indexnow yaml config, mapped to `--config`. Closes a gap from v0.5.0 — previously the Action had no way to point at a `.indexnow.yaml` checked into the repo.

[0.6.0]: https://github.com/jtprogru/indexnow/releases/tag/v0.6.0

## [0.5.0] — 2026-06-02

### Added

- Reusable GitHub Action: `jtprogru/indexnow@v0`. Composite action that downloads the goreleaser binary for the runner's OS/arch from the matching tag, sha256-verifies it against the release's `checksums.txt`, caches it in `runner.tool_cache` keyed by version-os-arch, and runs `indexnow submit` with the workflow inputs. Supports `ubuntu-*` and `macos-*` runners (x64 and arm64); Windows fails explicitly in preflight with a pointer to the supported runners. Inputs mirror `submit` flags 1:1 (`urls`/`file`/`sitemap`, `key`, `host`, `endpoint`, `fail-on`, retry knobs, `dry-run`, `verbose`, `quiet`, `user-agent`, …). Outputs: `exit-code`, `submitted-count`, `failed-count`, `report` (also written to `$GITHUB_STEP_SUMMARY`). Sensitive values (`key`, `key-location`) are passed via env, not flags, so `set -x` debugging cannot leak them. Companion workflow `release-major-alias.yaml` force-moves the floating `vX` and `vX.Y` tags on every `vX.Y.Z` release, so `@v0` always points to the latest release in that major.

[0.5.0]: https://github.com/jtprogru/indexnow/releases/tag/v0.5.0

## [0.4.0] — 2026-06-02

### Added

- `--sitemap <url|path>` for `submit`: a fourth URL source alongside positional args / `--file` / `--stdin`. Accepts either an absolute http(s) URL or a local filesystem path. `<sitemapindex>` documents are followed recursively (depth-capped, visited sources deduped), `.gz` sources are gunzipped transparently, and self-references terminate cleanly. Companion flags: `--sitemap-since <RFC3339>` filters entries by `<lastmod>` (entries without lastmod always pass — absent signal is treated as "may have changed", which is the safe default for IndexNow), and `--sitemap-timeout` caps per-request HTTP timeout (default `30s`). New package `internal/sitemap` parses the wire format namespace-agnostically via streaming `encoding/xml`, so 50 MB / 50 000-entry sitemaps don't load whole into memory.

[0.4.0]: https://github.com/jtprogru/indexnow/releases/tag/v0.4.0

## [0.3.0] — 2026-06-01

### Added

- `--verbose` / `-v` flag for `submit`: emit `slog` text-format lifecycle and retry events to stderr (`submit` start, per-batch `submit batch`, `retry` at WARN with status/transport reason and backoff, per-endpoint `batch complete`). stdout is untouched, so `-v` composes with `-q` and with `--output json` for log-shipping setups. By default the client uses `slog.DiscardHandler`, so library users and the no-flag CLI mode emit nothing.
- New subcommand `indexnow verify`: HTTP `GET` the hosted key file and check its trimmed body equals `--key`. URL is either `--key-location` (explicit) or derived from `--host` + `--key` (`https://<host>/<key>.txt`). Supports `--config`, `--user-agent`, `--output text|json`, `-q`, `-v`, `--timeout`. Exit `0` on match, `1` on mismatch/non-200/network error, `2` on usage error.

[0.3.0]: https://github.com/jtprogru/indexnow/releases/tag/v0.3.0

## [0.2.0] — 2026-06-01

### Added

- `--config <path>` flag for `submit`: load `host`, `key`, `key_location`, `endpoint`, `user_agent` defaults from a yaml file. Default lookup at `$XDG_CONFIG_HOME/indexnow/config.yaml` (fallback `$HOME/.config/indexnow/config.yaml`). Precedence: flag > env > config > built-in default. Unknown fields rejected.
- `--endpoint` accepts a comma-separated list and submits to every endpoint in parallel. Aliases and full URLs can be mixed; duplicates are removed while order is preserved. Single-endpoint output is unchanged; multi-endpoint text output prefixes each line with `endpoint=<url>`, JSON output emits one entry per endpoint × batch. Endpoint-level errors (factory init, transport failure) always produce a non-zero exit even under `--fail-on=never`.
- `--quiet` / `-q` flag for `submit`: suppress all stdout (both per-batch text and JSON), keeping the exit code as the only signal. Validation and system errors still go to stderr. Pairs naturally with `&&` / `||` in scripts.
- `--user-agent` flag and `INDEXNOW_USER_AGENT` env / `user_agent` config field for `submit`. HTTP requests now ship a `User-Agent` header instead of the stdlib default `Go-http-client/1.1`. Default value: `indexnow/<version>`; useful for WAF / proxy allow-lists and for endpoint-side logs that want to identify the caller.

[0.2.0]: https://github.com/jtprogru/indexnow/releases/tag/v0.2.0

## [0.1.0] — 2026-06-01

### Added

- Initial CLI: `indexnow submit` with positional / `--file` / `--stdin` URL sources.
- HTTP client with retry (429 / 5xx / transport errors), exponential backoff, jitter, and `Retry-After` support.
- Endpoint aliases: `api`, `bing`, `yandex`, `naver`, `seznam`, `yep`, plus pass-through for arbitrary URLs.
- Batching at the protocol limit (`MaxBatchSize = 10000`).
- Output formats: `text`, `json`. Exit policy: `--fail-on any|4xx|5xx|never`.
- ENV fallbacks: `INDEXNOW_KEY`, `INDEXNOW_HOST`, `INDEXNOW_KEY_LOCATION`, `INDEXNOW_ENDPOINT`.
- Project infrastructure: Taskfile, golangci-lint v2, GoReleaser, GitHub Actions (lint / tests / goreleaser / docs), Dependabot, bilingual MkDocs site.
- Homebrew cask in `jtprogru/homebrew-tap` published from the GoReleaser pipeline.

[0.1.0]: https://github.com/jtprogru/indexnow/releases/tag/v0.1.0
