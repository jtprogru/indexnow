# `indexnow verify`

Проверяет, что hosted key-файл сайта совпадает с ожидаемым ключом. Полезно перед первой отправкой и как smoke-проверка в deploy-пайплайне после ротации ключа.

## Использование

```bash
indexnow verify [flags]
```

Команда делает один HTTP `GET` по URL key-файла и сверяет trimmed body с `--key`. Без retry — это диагностика, single-shot.

## Резолв URL

- Если задан `--key-location`, дёргается этот URL.
- Иначе URL выводится по конвенции `https://<host>/<key>.txt` из `--host` и `--key`. В этом режиме оба обязательны, `--key` должен удовлетворять формату IndexNow (`[A-Za-z0-9-]{8,128}`).

## Флаги

| Флаг                                | Назначение                                                                 |
|-------------------------------------|----------------------------------------------------------------------------|
| `--key`                             | IndexNow-ключ (env: `INDEXNOW_KEY`)                                        |
| `--host`                            | Хост сайта (env: `INDEXNOW_HOST`)                                          |
| `--key-location`                    | Абсолютный URL key-файла (env: `INDEXNOW_KEY_LOCATION`)                    |
| `--config PATH`                     | Yaml-конфиг с дефолтами `host`/`key`/`key_location`/`user_agent`           |
| `--user-agent STRING`               | HTTP-заголовок `User-Agent` (env: `INDEXNOW_USER_AGENT`; default: `indexnow/<version>`) |
| `--timeout DURATION`                | HTTP-таймаут запроса (default `10s`)                                       |
| `--output text\|json`               | Формат вывода                                                              |
| `-q, --quiet`                       | Глушит stdout; результат — в exit-коде                                     |
| `-v, --verbose`                     | Лог жизненного цикла в stderr (slog text)                                  |

## Примеры

```bash
# URL по конвенции: https://example.com/<KEY>.txt
indexnow verify --host example.com --key abcdef1234567890

# Нестандартный путь хостинга
indexnow verify --key abcdef1234567890 \
  --key-location https://static.example.com/indexnow/key.txt

# В скрипте: молчит на успехе, ответ — в exit-коде
indexnow verify -q --host example.com --key abcdef1234567890 \
  && echo "key live" || echo "fix the key file"
```

## Вывод

В text-режиме одна строка:

```
OK:   https://example.com/abcdef1234567890.txt
FAIL: https://example.com/abcdef1234567890.txt status=404 err=http 404
FAIL: https://example.com/abcdef1234567890.txt status=200 err=hosted key does not match expected
```

JSON-режим эмитит один объект: `{ "url", "ok", "status", "hosted", "error" }`. Поле `hosted` заполняется при mismatch'е (обрезано до 80 символов) — чтобы глазами увидеть, что реально отдают.

## Коды выхода

- `0` — hosted-ключ совпал
- `1` — mismatch, non-200, или сетевая ошибка
- `2` — usage error (нет ключа, кривой `--key-location`, и т.д.)
