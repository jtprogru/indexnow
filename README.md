# indexnow

📚 **Documentation:** [jtprogru.github.io/indexnow](https://jtprogru.github.io/indexnow/) (EN + RU).

CLI-клиент протокола [IndexNow](https://www.indexnow.org/documentation) для контентных пайплайнов. Один HTTP-вызов уведомляет участвующие поисковики (Bing, Yandex, Naver, Seznam, Yep, ...) об изменении URL — по спецификации submission к любому эндпоинту шарится со всеми остальными участниками.

## Install

Через Homebrew (macOS / Linux):

```bash
brew tap jtprogru/tap
brew install indexnow
```

Через `go install`:

```bash
go install github.com/jtprogru/indexnow/cmd/indexnow@latest
```

Готовые бинарники под Linux / macOS / FreeBSD (amd64, arm64) — на странице [Releases](https://github.com/jtprogru/indexnow/releases).

## Uninstall

Homebrew:

```bash
brew uninstall indexnow
brew untap jtprogru/tap   # опционально
```

Установленный через `go install` или вручную:

```bash
rm "$(command -v indexnow)"
```

## Usage

```bash
indexnow --help
indexnow submit --help
```

Минимальный вызов — позиционный URL:

```bash
export INDEXNOW_KEY=8f7e6d5c4b3a29180706050403020100
indexnow submit https://example.com/posts/new
```

Из файла (по одному URL на строку, `#` — комментарий):

```bash
indexnow submit --file urls.txt --endpoint bing
```

Из stdin:

```bash
sitemap-to-urls | indexnow submit --stdin --output json
```

### Источники URL

Ровно один из: позиционные аргументы, `--file PATH`, `--stdin`.

### Эндпоинты

`--endpoint` принимает алиас или полный URL:

| Алиас    | URL                                          |
|----------|----------------------------------------------|
| `api`    | `https://api.indexnow.org/indexnow` (default) |
| `bing`   | `https://www.bing.com/indexnow`              |
| `yandex` | `https://yandex.com/indexnow`                |
| `naver`  | `https://searchadvisor.naver.com/indexnow`   |
| `seznam` | `https://search.seznam.cz/indexnow`          |
| `yep`    | `https://indexnow.yep.com/indexnow`          |

По спеке достаточно одного — submission шарится с остальными участниками.

### Окружение

| ENV                      | Назначение                                |
|--------------------------|-------------------------------------------|
| `INDEXNOW_KEY`           | ключ (валидируется по спеке: 8..128, `[A-Za-z0-9-]`) |
| `INDEXNOW_HOST`          | хост сайта (например `example.com`); если пуст — выводится из первого URL |
| `INDEXNOW_KEY_LOCATION`  | абсолютный URL к hosted key-файлу         |
| `INDEXNOW_ENDPOINT`      | алиас или URL эндпоинта                   |

### Поведение по ошибкам

`--fail-on any|4xx|5xx|never` — определяет, какие ответы поднимают exit code в 1. По умолчанию `any` (любая не-2xx или transport error → exit 1).

Коды выхода:

- `0` — OK
- `1` — submission failed (HTTP / network / fail-on triggered)
- `2` — usage error (неверные флаги, нет источника URL, нет ключа)

### Ретраи

Ретраит 429, 5xx и transport-ошибки с экспоненциальным backoff'ом и jitter'ом. Уважает заголовок `Retry-After` (и секунды, и HTTP-date). Настраивается флагами `--max-retries`, `--base-backoff`, `--max-backoff`.

### Батчинг

`SubmitBatch` автоматически разбивает входной список по `MaxBatchSize = 10000` (лимит протокола), отправляя несколько POST-запросов и возвращая по результату на батч.

## Development

```bash
task          # список задач
task build    # собрать в ./dist/indexnow
task test     # go test --short -coverprofile=cover.out -v ./...
task test:race
task lint     # golangci-lint run
task ci       # lint + race tests (квик локальный pre-push)
```

## License

[MIT](./LICENSE) © Mikhail Savin
