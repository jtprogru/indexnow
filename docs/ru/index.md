# indexnow

**indexnow** — компактный CLI для уведомления поисковиков об изменениях URL по протоколу [IndexNow](https://www.indexnow.org/documentation). Один HTTP-вызов к любому участнику (Bing, Yandex, Naver, Seznam, Yep, ...) — submission шарится со всеми остальными.

Сделан под контентные пайплайны: детерминированный, скриптуемый, явно падает на ошибках, без скрытого состояния.

---

## Что внутри

- **Одна подкоманда, три источника URL** — позиционные аргументы, `--file`, `--stdin`.
- **Шесть эндпоинтов из коробки** — `api`, `bing`, `yandex`, `naver`, `seznam`, `yep` — или произвольный URL.
- **Ретраи по спеке** — экспоненциальный backoff с jitter'ом, заголовок `Retry-After` (секунды + HTTP-date) на 429 / 5xx / transport-ошибках.
- **Батчинг по лимиту протокола** — автоматическое разбиение на 10 000 URL на POST.
- **Pipeline-friendly вывод** — `--output text|json`, политика выхода через `--fail-on any|4xx|5xx|never`.
- **ENV-fallback'и** — `INDEXNOW_KEY`, `INDEXNOW_HOST`, `INDEXNOW_KEY_LOCATION`, `INDEXNOW_ENDPOINT`.

## Быстрый старт

```bash
# Один URL, минимум флагов
export INDEXNOW_KEY=8f7e6d5c4b3a29180706050403020100
indexnow submit https://example.com/posts/new

# Массово из файла, JSON под jq
indexnow submit --file urls.txt --endpoint bing --output json | jq '.[].status'

# Из stdin через пайп
sitemap-to-urls | indexnow submit --stdin --fail-on 5xx
```

## Дальше

- **[Старт](getting-started.md)** — установка и первый вызов.
- **[Команды](commands/index.md)** — полный референс `indexnow submit`.
- **[Руководства](guides/endpoints.md)** — эндпоинты, конфигурация, ENV.
