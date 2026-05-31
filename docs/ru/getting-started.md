# Старт

## Установка

Homebrew (macOS / Linux):

```bash
brew tap jtprogru/tap
brew install indexnow
```

`go install`:

```bash
go install github.com/jtprogru/indexnow/cmd/indexnow@latest
```

Или готовый бинарь со страницы [Releases](https://github.com/jtprogru/indexnow/releases).

## Получите IndexNow-ключ

Протокол требует ключ (8..128 символов, `[A-Za-z0-9-]`), размещённый по известному URL на вашем сайте — обычно `https://example.com/<key>.txt`, где в файле — только этот ключ одной строкой. Подробнее — в [документации IndexNow](https://www.indexnow.org/documentation).

```bash
export INDEXNOW_KEY=8f7e6d5c4b3a29180706050403020100
```

## Первый вызов

```bash
indexnow submit https://example.com/posts/new
```

Готово. Host выводится из URL, дефолтный эндпоинт — `https://api.indexnow.org/indexnow`, и по спеке submission шарится со всеми поисковиками-участниками.

## Dry-run

```bash
indexnow submit --dry-run --file urls.txt
```

Покажет, что было бы отправлено, и выйдет с `0` — без сетевого вызова.
