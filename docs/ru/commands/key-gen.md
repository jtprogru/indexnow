# `indexnow key gen`

Сгенерировать случайный IndexNow-ключ (по дефолту 32 hex-символа, 128 бит энтропии) и опционально записать hosted key-файл `<dir>/<key>.txt` рядом.

## Synopsis

```bash
indexnow key gen [flags]
```

Одноразовая developer-time операция — обычно запускается один раз для сайта, ключ сохраняется в env / config / CI-secret и больше не трогается. Полный workflow — в [Жизненный цикл ключа](../guides/key-lifecycle.md).

## Флаги

| Флаг                          | Назначение                                                       |
|-------------------------------|------------------------------------------------------------------|
| `--length INT`                | Длина ключа в hex-символах (8..128). Default `32` (= 128 бит).   |
| `--write DIR`                 | Записать `<DIR>/<key>.txt` с правами `0644`. Директория должна существовать. |
| `--force`                     | Переписать существующий `<key>.txt` (по дефолту — отказ с exit 1). |
| `--output text\|json`         | Формат вывода. Default `text`.                                   |
| `-q, --quiet`                 | Заглушить stdout; полагаться на exit-code (файл всё равно создаётся). |

## Примеры

```bash
# Просто выдать ключ в stdout — pipe-friendly.
KEY=$(indexnow key gen)

# Сгенерировать и положить hosted-файл в исходники static-сайта.
indexnow key gen --write public/

# Машинно-читаемый вывод: и ключ, и путь.
indexnow key gen --write public/ --output json
# {"key":"3f0e2e5bb2675d1742842df41986a2f1","path":"public/3f0e2e5bb2675d1742842df41986a2f1.txt"}

# Сгенерировать в не-корневую hosting-локацию.
indexnow key gen --write public/.well-known/indexnow/

# Quiet bootstrap-скрипт — только side-effect (файл + exit-code).
indexnow key gen --write public/ -q
```

## Вывод

`--output text` (default): ключ одной строкой в stdout. С `--write` в stderr печатается notice `wrote <path>`.

`--output json`: один объект в stdout.

```json
{"key":"...","path":"public/...txt"}
```

Поле `path` отсутствует, когда `--write` не указан.

## Exit-коды

- `0` — ключ сгенерирован (и файл записан, если `--write`).
- `1` — I/O-ошибка (директория не существует, файл уже есть и нет `--force`, системный CSPRNG упал).
- `2` — usage error (`--length` вне диапазона, неизвестное значение `--output`).

## Заметки

- **Имя файла должно совпадать с ключом.** Если перенаправить stdout в файл руками (`indexnow key gen > public/mykey.txt`), поисковики его не найдут — они ищут `<key>.txt`. Используйте `--write` для файловой формы; имя строится из ключа автоматически.
- **Trailing newline.** Hosted-файл пишется как `<key>\n`. Все известные IndexNow-эндпоинты тримят whitespace перед сравнением; newline — unix-конвенция.
- **Не для CI.** `indexnow key gen` — одноразовый bootstrap-шаг, не per-push операция. Запуск в workflow перегенерил бы ключ на каждый build и инвалидировал submission'ы между прогонами. IndexNow Action осознанно не экспозит генерацию ключа как input — см. [Жизненный цикл ключа](../guides/key-lifecycle.md#4-wire-it-up--где-ключ-живт-в-момент-использования).
- **Источник энтропии.** Только `crypto/rand`. Публичного `--seed` нет; тесты подменяют источник через package-private hook.
- **`--write` не делает `mkdir -p`.** Если директория не существует, `key gen` падает. Создайте директорию заранее.
