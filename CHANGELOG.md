# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The GoReleaser pipeline auto-generates per-release notes on the GitHub Releases page from commit messages; this file is the project-level human-curated history.

## [Unreleased]

### Added

- `--config <path>` flag for `submit`: load `host`, `key`, `key_location`, `endpoint` defaults from a yaml file. Default lookup at `$XDG_CONFIG_HOME/indexnow/config.yaml` (fallback `$HOME/.config/indexnow/config.yaml`). Precedence: flag > env > config > built-in default. Unknown fields rejected.
- `--endpoint` accepts a comma-separated list and submits to every endpoint in parallel. Aliases and full URLs can be mixed; duplicates are removed while order is preserved. Single-endpoint output is unchanged; multi-endpoint text output prefixes each line with `endpoint=<url>`, JSON output emits one entry per endpoint × batch. Endpoint-level errors (factory init, transport failure) always produce a non-zero exit even under `--fail-on=never`.
- `--quiet` / `-q` flag for `submit`: suppress all stdout (both per-batch text and JSON), keeping the exit code as the only signal. Validation and system errors still go to stderr. Pairs naturally with `&&` / `||` in scripts.
- `--user-agent` flag and `INDEXNOW_USER_AGENT` env / `user_agent` config field for `submit`. HTTP requests now ship a `User-Agent` header instead of the stdlib default `Go-http-client/1.1`. Default value: `indexnow/<version>`; useful for WAF / proxy allow-lists and for endpoint-side logs that want to identify the caller.

### Changed

-

### Fixed

-

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
