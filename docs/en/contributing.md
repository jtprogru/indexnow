# Contributing

PRs welcome. The project is small enough that there is no ceremony.

## Local development

```bash
git clone https://github.com/jtprogru/indexnow
cd indexnow
task          # list available targets
task build    # binary into ./dist
task ci       # lint + race tests — what CI runs
```

## Style

- `gofmt -s` (run via `task fmt`).
- `golangci-lint` config in `.golangci.yaml`. Run `task lint`.
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
