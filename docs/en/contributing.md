# Contributing

PRs welcome. The project is small enough that there is no ceremony.

## Local development

```bash
git clone https://github.com/jtprogru/indexnow
cd indexnow
make          # list available targets
make build    # binary into ./dist
make ci       # lint + race tests — what CI runs
```

CI calls the same targets (`make test-race`, `make docs-install`, `make docs-deploy`) rather than repeating the commands inline, so a green `make ci` locally means the same thing it means in the workflow.

## Style

- `gofmt -s` (run via `make fmt`).
- `golangci-lint` config in `.golangci.yaml`. Run `make lint`.
- Tests run with `-race` in CI; keep them passing under the race detector.

## Commit messages

Conventional-ish but not enforced. Dependabot uses `chore(deps):` and `chore(ci):` prefixes — feel free to match.

## Docs

The site is MkDocs Material with EN and RU built from `docs/en` and `docs/ru`. Both languages are expected to stay in sync — a change to one without the other is an incomplete change.

```bash
make docs-install   # pip install -r docs/requirements.txt
make docs-serve     # http://127.0.0.1:8000
make docs-build     # mkdocs build --strict
```

A push to `main` touching `docs/**` or `mkdocs.yml` deploys to GitHub Pages.

## Releasing

Maintainers cut the CHANGELOG entry first, then tag from `main` with an annotated, GPG-signed tag:

```bash
git tag -s vX.Y.Z -m "release vX.Y.Z"
git push origin vX.Y.Z
```

The tag push triggers two workflows:

- `goreleaser` — builds Linux / macOS / FreeBSD binaries for amd64 and arm64, GPG-signs `checksums.txt`, publishes the GitHub release, and updates the Homebrew cask in `jtprogru/homebrew-tap`.
- `release-major-alias` — force-moves the floating `vX` and `vX.Y` tags onto the new release, so `jtprogru/indexnow@v0` always resolves to the newest `v0.*`.

If the release job fails partway and you re-run it, note that goreleaser is configured with `release.replace_existing_artifacts: true` — otherwise a retry on an already-published tag dies with `422 already_exists` on every asset.
