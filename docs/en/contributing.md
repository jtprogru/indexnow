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

## Releasing

Maintainers tag from `main`:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

GoReleaser builds binaries, signs the checksum, and updates the Homebrew tap.
