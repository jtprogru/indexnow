# `indexnow key gen`

Generate a random IndexNow key (32 hex chars by default, 128 bits of entropy), optionally writing the hosted key file `<dir>/<key>.txt` alongside it.

## Synopsis

```bash
indexnow key gen [flags]
```

This is a one-shot, developer-time operation — you typically run it once per site, store the resulting key wherever it lives (env / config / CI secret), and never touch it again. See [Key lifecycle](../guides/key-lifecycle.md) for the wider workflow.

## Flags

| Flag                          | Purpose                                                          |
|-------------------------------|------------------------------------------------------------------|
| `--length INT`                | Key length in hex chars (8..128). Default `32` (= 128 bits).     |
| `--write DIR`                 | Write `<DIR>/<key>.txt` with mode `0644`. Directory must exist.  |
| `--force`                     | Overwrite an existing `<key>.txt` (default: refuse with exit 1). |
| `--output text\|json`         | Output format. Default `text`.                                   |
| `-q, --quiet`                 | Suppress stdout; rely on exit code (file is still written).      |

## Examples

```bash
# Just emit a key to stdout — pipe-friendly.
KEY=$(indexnow key gen)

# Generate and drop the hosted file into your static-site source.
indexnow key gen --write public/

# Machine-readable, both the key and the path it landed at.
indexnow key gen --write public/ --output json
# {"key":"3f0e2e5bb2675d1742842df41986a2f1","path":"public/3f0e2e5bb2675d1742842df41986a2f1.txt"}

# Generate into a non-root hosting location.
indexnow key gen --write public/.well-known/indexnow/

# Quiet bootstrap script — only side-effects (file + exit code).
indexnow key gen --write public/ -q
```

## Output

`--output text` (default): the key, one line, to stdout. With `--write`, a `wrote <path>` notice is printed to stderr.

`--output json`: a single object on stdout.

```json
{"key":"...","path":"public/...txt"}
```

`path` is omitted from the JSON when `--write` is not used.

## Exit codes

- `0` — key generated (and file written, if `--write`).
- `1` — I/O failure (write target is a non-existent directory, the file already exists without `--force`, the system entropy source failed).
- `2` — usage error (`--length` out of range, unknown `--output` value).

## Notes

- **Filename must match the key.** If you redirect stdout to a file by hand (`indexnow key gen > public/mykey.txt`), search engines won't find it — they look for `<key>.txt`. Use `--write` for the file form; the name is built from the key automatically.
- **Trailing newline.** The hosted file is written as `<key>\n`. All known IndexNow endpoints trim whitespace before comparing; the newline is unix-conventional.
- **Not for CI.** `indexnow key gen` is a one-time bootstrap step, not a per-push operation. Running it in a workflow would regenerate the key on every build, invalidating submissions between runs. The IndexNow Action deliberately does not expose key generation as an input — see [Key lifecycle](../guides/key-lifecycle.md#4-wire-it-up--where-the-key-lives-at-use-time).
- **Source of entropy.** `crypto/rand` only. There is no `--seed` flag; tests substitute the source via a package-private hook.
- **`--write` does not `mkdir -p`.** If the target directory doesn't exist, `key gen` fails. Create it first.
