# Configuration

Configuration is flag-first with ENV and an optional yaml file as fallbacks. Secrets stay where you put them — environment, secret manager, or a per-user file.

## Precedence

1. CLI flag (e.g. `--key`)
2. Environment variable
3. yaml config file
4. Built-in default (where applicable)

## Environment variables

| ENV                      | Maps to             |
|--------------------------|---------------------|
| `INDEXNOW_KEY`           | `--key`             |
| `INDEXNOW_HOST`          | `--host`            |
| `INDEXNOW_KEY_LOCATION`  | `--key-location`    |
| `INDEXNOW_ENDPOINT`      | `--endpoint`        |

`INDEXNOW_ENDPOINT` only takes effect when `--endpoint` is left at its default (`api`); explicit flag overrides ENV.

## Config file

Pass an explicit path via `--config <file>`. Without the flag, indexnow looks at `$XDG_CONFIG_HOME/indexnow/config.yaml` (or `$HOME/.config/indexnow/config.yaml`); a missing default file is silently skipped, a missing explicit `--config` path is a usage error.

Schema (all keys optional):

```yaml
host: example.com
key: abc123
key_location: https://example.com/abc123.txt
endpoint: bing
```

Unknown keys are rejected so typos surface immediately. Empty file is treated as no config.

## Host inference

If `--host` / `INDEXNOW_HOST` / config `host` are all empty, the host is parsed from the **first** URL in the batch. All URLs in one call must belong to the same site per the protocol spec.
