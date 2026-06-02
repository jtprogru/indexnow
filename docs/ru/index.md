# indexnow

**indexnow** уведомляет поисковики (Bing, Yandex, Naver, Seznam, Yep, ...) об изменениях URL по протоколу [IndexNow](https://www.indexnow.org/documentation). Один HTTP-вызов — submission по спеке шарится со всеми участниками, никакой per-engine обвязки.

Сделан под контентные пайплайны: детерминированный, скриптуемый, явно падает на ошибках, без скрытого состояния. Поставляется как **GitHub Action** для CI и **CLI** для всего остального.

---

## Рекомендуемый способ: GitHub Action

```yaml
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    sitemap: https://example.com/sitemap.xml
```

Один шаг в любом workflow. Без `setup-go`, без Docker pull — action скачивает запиннутый релизный бинарь под OS/arch раннера, sha256-проверяет, кэширует, и запускает `indexnow submit` с вашими inputs. Чувствительные значения идут через env, а не через флаги командной строки.

→ Полный референс inputs/outputs и рецепты CI: **[GitHub Action](guides/github-action.md)**.

## Альтернатива: CLI

Для локальных запусков, cron-скриптов и других CI:

```bash
brew install jtprogru/tap/indexnow

export INDEXNOW_KEY=8f7e6d5c4b3a29180706050403020100
indexnow submit https://example.com/posts/new

# массово из sitemap
indexnow submit --sitemap https://example.com/sitemap.xml --sitemap-since 2026-05-01T00:00:00Z

# fan-out на несколько эндпоинтов, JSON под downstream-тулинг
indexnow submit --file urls.txt --endpoint bing,yandex --output json | jq '.[].status'
```

→ Установка, первый вызов, dry-run: **[Старт](getting-started.md)**.

## Что внутри

- **Четыре источника URL** — позиционные аргументы, `--file`, `--stdin`, `--sitemap` (с `.gz` и `<sitemapindex>`, RFC3339-фильтр `--sitemap-since`).
- **Шесть эндпоинтов из коробки** — `api`, `bing`, `yandex`, `naver`, `seznam`, `yep` — или произвольный URL. Comma-separated список — параллельный fan-out.
- **Ретраи по спеке** — экспоненциальный backoff с jitter'ом, заголовок `Retry-After` (секунды + HTTP-date) на 429 / 5xx / transport-ошибках.
- **Батчинг по лимиту протокола** — автоматическое разбиение по 10 000 URL на POST.
- **Pipeline-friendly вывод** — `--output text|json`, политика выхода через `--fail-on any|4xx|5xx|never`, `--quiet` — exit-код как единственный сигнал.
- **Наблюдаемость без подвохов** — `--verbose` пишет `slog` lifecycle/retry в stderr, stdout остаётся чистым для downstream-тулинга.

## Дальше

- **[GitHub Action](guides/github-action.md)** — inputs, outputs, рецепты CI.
- **[Старт](getting-started.md)** — установка CLI и первый вызов.
- **[CLI-команды](commands/index.md)** — референс `submit` и `verify`.
- **[Руководства](guides/endpoints.md)** — эндпоинты, конфигурация, ENV.
