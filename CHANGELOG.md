# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The GoReleaser pipeline auto-generates per-release notes on the GitHub Releases page from commit messages; this file is the project-level human-curated history.

## [Unreleased]

### Added

-

### Changed

-

### Fixed

-

## [0.7.2] — 2026-08-10

Documentation and tooling release: no behavior changes in the CLI or the Action. Two of the fixes below are user-facing all the same — a published Action recipe did not work when copied, and in-page links on the RU documentation site did not resolve.

### Changed

- Task runner switched from `Taskfile.yml` to a `Makefile`. Targets carry over one-to-one with `:` replaced by `-` (`task test:race` → `make test-race`, `task docs:build` → `make docs-build`); `make` with no argument prints the self-documenting target list. New `docs-deploy` (the `mkdocs gh-deploy` invocation CI was running inline) and `clean` targets. CI now calls the targets instead of repeating commands: `tests.yaml` runs `make test-race`, `docs.yaml` runs `make docs-install` + `make docs-deploy`. `lint.yaml` and `goreleaser.yaml` keep their respective actions — those handle tool install and caching and never duplicated a command in the first place.
- Docs no longer name a concrete release. The exact-pin example in the Action guide used a real tag, which meant every release needed a documentation edit to stay truthful; it now shows the `@vX.Y.Z` placeholder and links to Releases. Everything else already used the floating `@v0`, which the alias workflow moves on each release. `contributing.md` records the convention so it does not creep back.
- Docs (EN+RU) brought back in line with the code. `architecture.md` now shows the real tree — `internal/cli/keygen.go`, `internal/cli/verify.go`, `internal/config`, `internal/sitemap`, `action.yml` and `scripts/` were all missing — and documents what those packages own plus how endpoint fan-out is scheduled. `contributing.md` documents the release flow as it actually runs (annotated GPG-signed tag, the `goreleaser` and `release-major-alias` workflows, the `replace_existing_artifacts` retry caveat) and adds the MkDocs targets. `README.md` gained the sections it never had: comma-separated multi-endpoint fan-out, the yaml config file and `--config`, `INDEXNOW_USER_AGENT`, and the `--output` / `-q` / `-v` interplay.
- `key-lifecycle.md` (EN+RU) no longer implies a repo-local `.indexnow.yaml` is discovered automatically. Only `$XDG_CONFIG_HOME/indexnow/config.yaml` is read on its own; a project config has to be passed via `--config` (CLI) or the Action's `config:` input.

### Fixed

- Docs recipe `sitemap-since: ${{ github.event.before }}` was broken: `github.event.before` is a commit SHA, not a timestamp, and `--sitemap-since` parses RFC3339 only — copying the recipe failed the step with exit `2`. The push and schedule recipes now submit the whole sitemap by default (IndexNow is idempotent), with a separate block showing how to derive a real RFC3339 cutoff via `git show -s --format=%cI` or `date -u`. Affected `guides/github-action.md` and `getting-started.md` in both languages.
- Anchors on the RU docs site. Python-Markdown's default slugify NFKD-normalizes headings and drops non-ASCII, so every Cyrillic heading collapsed to a truncated or empty id (`Типовые грабли` → `""`, `Кастомные источники URL через urls-from` → `url-urls-from`), breaking in-page links and permalinks across the whole RU site. `mkdocs.yml` now uses `pymdownx.slugs.slugify(case="lower")`, which preserves Unicode; ASCII headings slugify identically, so EN anchors are unchanged. The two RU cross-links that pointed at hand-written slugs were corrected to match.

[0.7.2]: https://github.com/jtprogru/indexnow/releases/tag/v0.7.2

## [0.7.1] — 2026-08-10

Maintenance release: no behavior changes in the CLI or the Action, dependency and toolchain refresh only.

### Changed

- `actions/cache` in the composite action bumped `v4` → `v6`. This is the only change that touches `action.yml`; the cache key layout (`indexnow-<version>-<os>-<arch>`) is unchanged, so existing caches keep working and consumers of `jtprogru/indexnow@v0` need no action.
- CI workflows moved to the current major of every action in use: `actions/checkout` `v6` → `v7`, `actions/setup-go` `v6` → `v7`, `actions/setup-python` `v6` → `v7`, `codecov/codecov-action` `v6` → `v7`, and the pinned `goreleaser/goreleaser-action` digest updated to `v7.2.3`.
- `go.yaml.in/yaml/v3` bumped `3.0.4` → `3.0.5`.
- Added a `markdownlint` config so docs and changelog lint consistently across local runs and CI.

[0.7.1]: https://github.com/jtprogru/indexnow/releases/tag/v0.7.1

## [0.7.0] — 2026-06-02

### Added

- New `indexnow key` command namespace that groups key-management operations. `indexnow key gen` generates a random hex-encoded IndexNow key (default 32 chars, 128 bits of entropy from `crypto/rand`; `--length 8..128` accepted) and optionally writes the hosted key file `<dir>/<key>.txt` with `--write <dir>` (mode `0644`, refuses to overwrite without `--force`). Output is the bare key on stdout for `KEY=$(indexnow key gen)`-style use; `--output json` emits `{"key":"…","path":"…"}`; `-q` suppresses stdout while still writing the file. `indexnow key verify` is the canonical form of the existing verify operation — same flags, same behavior. No new Action input: key management is intentionally CLI-only; the Action uses keys but does not generate or rotate them.
- New documentation page `guides/key-lifecycle.md` (EN+RU) walks through the whole flow — generate, deploy, hosting-mode choice, where the key lives at use-time, verify the bootstrap, use, manual rotate. Getting Started now defers to this guide instead of duplicating partial fragments.

### Changed

- `indexnow verify` (top-level) is kept as a backwards-compatibility alias for `indexnow key verify`. Existing scripts written against v0.3.0–v0.6.x continue to work unchanged. New code should prefer the canonical `key verify` form; the alias is documented in the command reference.

[0.7.0]: https://github.com/jtprogru/indexnow/releases/tag/v0.7.0

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
