# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The GoReleaser pipeline auto-generates per-release notes on the GitHub Releases page from commit messages; this file is the project-level human-curated history.

## [Unreleased]

### Added

- Initial CLI: `indexnow submit` with positional / `--file` / `--stdin` URL sources.
- HTTP client with retry (429 / 5xx / transport errors), exponential backoff, jitter, and `Retry-After` support.
- Endpoint aliases: `api`, `bing`, `yandex`, `naver`, `seznam`, `yep`, plus pass-through for arbitrary URLs.
- Batching at the protocol limit (`MaxBatchSize = 10000`).
- Output formats: `text`, `json`. Exit policy: `--fail-on any|4xx|5xx|never`.
- ENV fallbacks: `INDEXNOW_KEY`, `INDEXNOW_HOST`, `INDEXNOW_KEY_LOCATION`, `INDEXNOW_ENDPOINT`.
- Project infrastructure: Taskfile, golangci-lint v2, GoReleaser, GitHub Actions (lint / tests / goreleaser / docs), Dependabot, bilingual MkDocs site.

### Changed

-

### Fixed

-
