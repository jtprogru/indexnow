# Конфигурация

Конфигурация — флаги, с fallback'ом на ENV и опциональный yaml-файл. Секреты живут там, где их положили: окружение, менеджер секретов или пользовательский файл.

## Приоритеты

1. CLI-флаг (например `--key`)
2. Переменная окружения
3. yaml-конфиг
4. Встроенный дефолт (где применимо)

## Переменные окружения

| ENV                      | Маппится на         |
|--------------------------|---------------------|
| `INDEXNOW_KEY`           | `--key`             |
| `INDEXNOW_HOST`          | `--host`            |
| `INDEXNOW_KEY_LOCATION`  | `--key-location`    |
| `INDEXNOW_ENDPOINT`      | `--endpoint`        |
| `INDEXNOW_USER_AGENT`    | `--user-agent`      |

`INDEXNOW_ENDPOINT` срабатывает только если `--endpoint` оставлен в дефолте (`api`); явный флаг побеждает ENV.

## Файл конфига

Явный путь — `--config <file>`. Без флага indexnow смотрит в `$XDG_CONFIG_HOME/indexnow/config.yaml` (или `$HOME/.config/indexnow/config.yaml`); отсутствие дефолтного файла — это не ошибка, отсутствие явного `--config` — ошибка использования.

Схема (все ключи опциональны):

```yaml
host: example.com
key: abc123
key_location: https://example.com/abc123.txt
endpoint: bing
user_agent: my-pipeline/2.3
```

Неизвестные ключи отвергаются, чтобы опечатки всплывали сразу. Пустой файл = отсутствие конфига.

## Вывод host

Если `--host` / `INDEXNOW_HOST` / `host` из конфига — все пустые, host парсится из **первого** URL в батче. По спеке все URL одного вызова должны принадлежать одному сайту.
