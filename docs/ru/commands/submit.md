# `indexnow submit`

Отправляет один или несколько URL на IndexNow-эндпоинт.

## Использование

```bash
indexnow submit [urls...] [flags]
```

Ровно один источник URL:

- позиционные аргументы: `indexnow submit URL [URL ...]`
- `--file PATH` — по одному URL на строку, `#` — комментарии, пустые строки игнорируются
- `--stdin` — URL'ы из stdin

## Флаги

| Флаг                                | Назначение                                                                 |
|-------------------------------------|----------------------------------------------------------------------------|
| `--key`                             | IndexNow-ключ (env: `INDEXNOW_KEY`)                                        |
| `--host`                            | Хост сайта, например `example.com` (env: `INDEXNOW_HOST`; иначе выводится из URL) |
| `--key-location`                    | Абсолютный URL к hosted key-файлу (env: `INDEXNOW_KEY_LOCATION`)           |
| `--endpoint`                        | Один алиас/URL либо список через запятую (параллельный fan-out)             |
| `--config PATH`                     | Yaml-конфиг с дефолтами `host`/`key`/`key_location`/`endpoint`             |
| `--file PATH`                       | Читать URL из файла                                                        |
| `--stdin`                           | Читать URL из stdin                                                        |
| `--dry-run`                         | Показать, что было бы отправлено, и выйти                                  |
| `--output text\|json`               | Формат вывода                                                              |
| `-q, --quiet`                       | Глушит stdout; результат — только в exit-коде (ошибки идут в stderr)        |
| `--fail-on any\|4xx\|5xx\|never`    | Какие ответы поднимают exit code в 1                                       |
| `--max-retries N`                   | Ретраи на 429 / 5xx / transport-ошибках (default `3`)                      |
| `--base-backoff DURATION`           | Базовый backoff (default `1s`)                                             |
| `--max-backoff DURATION`            | Максимальный backoff (default `30s`)                                       |

## Примеры

```bash
indexnow submit https://example.com/post/1
indexnow submit --file urls.txt --endpoint bing
cat urls.txt | indexnow submit --stdin --output json
indexnow submit --endpoint bing,yandex https://example.com/post/1
indexnow submit -q https://example.com/post/1 && echo ok || echo failed
```

## Коды выхода

- `0` — OK
- `1` — submission failed (HTTP / network / fail-on)
- `2` — usage error
