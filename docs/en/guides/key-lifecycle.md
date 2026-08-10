# Key lifecycle

The IndexNow protocol uses a hosted-file key as proof you own the site you're submitting URLs for. This page walks through the whole lifecycle — generate, deploy, wire up, verify, use, rotate — so the moving parts make sense together.

If you only need a quick refresher: `indexnow key gen --write public/`, commit the resulting `<key>.txt`, deploy, then `indexnow submit ...`. The rest of this page is the "why" and the corners worth knowing.

## Why a key

IndexNow doesn't require an account anywhere. Instead, every submission you send carries:

- A `key` — the secret value.
- A `keyLocation` — a URL where that same value is served by your site.

Search engines fetch `keyLocation` over HTTPS, read the body, trim whitespace, and compare. Match → your submission is trusted as coming from someone who controls that domain. Mismatch or 404 → silently dropped.

So a key is, effectively, "a token you put on your own site to prove you can put files on your own site." Generated once, used many times.

## 1. Bootstrap — generate the key

```bash
indexnow key gen --write public/
```

What happens:

- 32 random hex characters are drawn from `crypto/rand` (128 bits of entropy).
- A file `public/<key>.txt` is created with permissions `0644` and a single line of content: the key.
- The key is printed to stdout; `wrote public/<key>.txt` goes to stderr.

Pipe-friendly:

```bash
KEY=$(indexnow key gen --write public/)
```

The `public/` directory must already exist — `indexnow` doesn't run `mkdir -p`. Use `--force` if you need to overwrite an existing file with the same name (rare; happens when re-running keygen with seeded entropy in tests).

## 2. Deploy — make the file live

Commit `<key>.txt` to your repo and push. After your site builds and deploys, the file must be reachable at the URL where IndexNow expects it (next section).

Three things have to be true for IndexNow to accept it:

- HTTPS (`https://`, not `http://`).
- HTTP 200 OK.
- Response body, trimmed of leading/trailing whitespace, equals the key exactly.

Static-site generators usually treat files in `public/`, `static/`, or `content/` as pass-through assets — drop the file in the matching directory for your generator and you're done.

## 3. Two hosting modes

There are two conventions for where the file lives. Pick one and stick with it.

### Default: `https://<host>/<key>.txt`

If you don't pass `--key-location` anywhere, both indexnow and the search engines assume the file sits at the root of your domain. This is the simplest path; no further configuration needed.

```bash
# bootstrap
indexnow key gen --write public/

# use
indexnow submit https://example.com/posts/foo --host example.com --key $KEY
```

Search engines fetch `https://example.com/<key>.txt`.

### Explicit `keyLocation`

You can also host the file anywhere on your domain — useful when the root is busy and you want all third-party verification files under one path:

```bash
indexnow key gen --write public/.well-known/indexnow/
indexnow submit https://example.com/posts/foo \
  --host example.com --key $KEY \
  --key-location https://example.com/.well-known/indexnow/$KEY.txt
```

Trade-off: every submission has to carry the explicit `--key-location` (or have it set via env / config). With the default mode, that's derived automatically.

Decide once, put `key_location:` in your repo's `.indexnow.yaml`, and point at it with `--config .indexnow.yaml` (CLI) or the action's `config:` input — see below.

## 4. Wire it up — where the key lives at use-time

The key shows up in three orthogonal places. Each is for a different audience.

| Where | Audience | Mechanism |
|---|---|---|
| `INDEXNOW_KEY` env var | Local shell, ad-hoc CLI runs | `export INDEXNOW_KEY=...` |
| `~/.config/indexnow/config.yaml` | Developer machine, persistent | YAML `key:` field — the only config path read automatically |
| `.indexnow.yaml` in repo | Project-level defaults (host, endpoint) | YAML; requires `--config` / `config:`. **Do not commit `key:` to a public repo** |
| GitHub Actions secret | CI workflow | `${{ secrets.INDEXNOW_KEY }}` → action `key:` input |

Precedence (most-specific wins): CLI flag > environment > config file > built-in default.

Only `$XDG_CONFIG_HOME/indexnow/config.yaml` (falling back to `$HOME/.config/indexnow/config.yaml`) is picked up on its own. A `.indexnow.yaml` sitting in your repo is **not** auto-discovered — pass it explicitly:

```bash
indexnow submit --config .indexnow.yaml https://example.com/posts/foo
```

For a typical CI setup:

1. Store the key as a repository secret named `INDEXNOW_KEY`.
2. Keep non-secret defaults (`host`, `endpoint`, `user_agent`) in a checked-in `.indexnow.yaml`.
3. Point the action at it with `config: .indexnow.yaml` (path is relative to `$GITHUB_WORKSPACE`, so the job needs `actions/checkout`).
4. Pass the secret into the action's `key:` input; the action plumbs it through as `INDEXNOW_KEY` env.

## 5. Verify — make sure the bootstrap worked

After the first deploy, run once:

```bash
indexnow key verify --host example.com --key $KEY
```

Exit `0` means the hosted file fetched cleanly and matches the key. Exit `1` means mismatch, non-200, or network error — the search engines will silently reject your submissions until you fix the hosting. `--verbose` shows the full URL and HTTP status.

You don't need to run `verify` on every push — it's a one-shot bootstrap check. Re-run it if you see unexpected 4xx counts in IndexNow responses, after redeploying with significant infrastructure changes, or whenever a teammate suspects the hosting drifted.

## 6. Use — submit URLs

Either path works:

```bash
# CLI
indexnow submit https://example.com/posts/foo

# GitHub Action
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    sitemap: https://example.com/sitemap.xml
```

See [CLI commands → submit](../commands/submit.md) and [GitHub Action](github-action.md) for the full surface.

## 7. Rotation (manual, for now)

`indexnow key rotate` is on the roadmap. Until it lands, rotate by hand when you need to — most projects never need to:

1. Generate a new key alongside the old one: `indexnow key gen --write public/`. Both `<old>.txt` and `<new>.txt` are now live.
2. Update wherever the key is stored (env / config / GitHub secret) to the new value. Submissions start carrying the new key immediately.
3. Wait. Search engines cache `keyLocation` aggressively; give them 1–2 weeks of grace so any in-flight verifications against the old key still resolve.
4. Delete `<old>.txt` from your repo and deploy.

The grace period is the awkward part — it's why `key rotate` is a v0.8 problem worth doing properly rather than as a one-line shell helper.

## Common pitfalls

- **Filename ≠ key.** If you write the file by hand instead of using `--write`, make sure the filename (minus `.txt`) is exactly the key. Mismatched names are the single most common verify failure.
- **Trailing newline.** `indexnow key gen --write` adds one. All known IndexNow endpoints trim whitespace before comparing, but if you author the file manually, a leading/trailing `\n` is fine — embedded whitespace or BOMs are not.
- **HTTP, not HTTPS.** A redirect from `http://` to `https://` works for browsers but some IndexNow verifiers don't follow it. Host the file at the canonical HTTPS URL directly.
- **Robots.txt or auth in front of the file.** The endpoint must serve 200 to anonymous clients without rate-limiting.
- **`--write` into a non-existent directory.** `indexnow` refuses by design. Create the directory first.
