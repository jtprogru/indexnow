# Configuration

Configuration is flag-first with ENV fallbacks. There is no config file — keep secrets in your environment or your secret manager.

## Precedence

1. CLI flag (e.g. `--key`)
2. Environment variable
3. Built-in default (where applicable)

## Environment variables

| ENV                      | Maps to             |
|--------------------------|---------------------|
| `INDEXNOW_KEY`           | `--key`             |
| `INDEXNOW_HOST`          | `--host`            |
| `INDEXNOW_KEY_LOCATION`  | `--key-location`    |
| `INDEXNOW_ENDPOINT`      | `--endpoint`        |

`INDEXNOW_ENDPOINT` only takes effect when `--endpoint` is left at its default (`api`); explicit flag overrides ENV.

## Host inference

If `--host` / `INDEXNOW_HOST` is empty, the host is parsed from the **first** URL in the batch. All URLs in one call must belong to the same site per the protocol spec.
