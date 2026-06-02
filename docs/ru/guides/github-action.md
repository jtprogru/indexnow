# GitHub Action

Используйте indexnow как шаг в любом workflow GitHub Actions. Action скачивает релизный бинарь под OS/arch раннера, сверяет sha256 с `checksums.txt` из того же релиза и запускает `indexnow submit` с переданными inputs.

```yaml
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    sitemap: https://example.com/sitemap.xml
```

Пинуйте на мажор (`@v0`) для floating-but-stable, на тег (`@v0.5.0`) для точности, на SHA коммита — для параноидального supply-chain режима.

## Поддерживаемые раннеры

| `runs-on` | Статус |
|---|---|
| `ubuntu-latest`, `ubuntu-*` (x64, arm64) | поддерживается |
| `macos-latest`, `macos-*` (x64, arm64) | поддерживается |
| `windows-*` | не поддерживается — preflight падает с подсказкой про ubuntu/macos |

## Inputs

| Имя | Обязательность | Default | Заметки |
|---|---|---|---|
| `key` | да | — | Ключ IndexNow (8..128, `[A-Za-z0-9-]`). Идёт через env, не флаги. |
| `urls` | один из | — | URL'ы через перевод строки. |
| `file` | один из | — | Путь к файлу (по одному URL на строку). |
| `sitemap` | один из | — | URL или локальный путь до `sitemap.xml` (`.gz` и `<sitemapindex>` раскрываются). |
| `sitemap-since` | нет | — | RFC3339; записи со старее `<lastmod>` отбрасываются. |
| `sitemap-timeout` | нет | дефолт CLI (`30s`) | Per-request HTTP timeout для sitemap. |
| `host` | нет | вывод из первого URL | Хост сайта (например `example.com`). |
| `key-location` | нет | derive из `host` + `key` | Абсолютный URL hosted-key файла. |
| `endpoint` | нет | `api` | Алиас или полный URL; comma-separated для fan-out. |
| `user-agent` | нет | `indexnow/<version>` | Заголовок `User-Agent`. |
| `fail-on` | нет | `any` | `any\|4xx\|5xx\|never` — какие классы ответа дают exit ≠ 0. |
| `quiet` | нет | `false` | Глушит stdout CLI в логе шага. |
| `verbose` | нет | `false` | slog lifecycle/retry в stderr. |
| `dry-run` | нет | `false` | Печатает план и выходит, не идёт по сети. |
| `max-retries` | нет | дефолт CLI | Ретраи на 429/5xx/transport. |
| `base-backoff` | нет | дефолт CLI | Начальный backoff (Go duration, например `200ms`). |
| `max-backoff` | нет | дефолт CLI | Кап backoff'а. |
| `version` | нет | тег action'а | Какой релиз indexnow ставить. `"latest"` резолвится в рантайме. |
| `github-token` | нет | `${{ github.token }}` | Используется только для GitHub Releases API. |

Ровно один из `urls` / `file` / `sitemap` должен быть задан — иначе preflight падает.

## Outputs

| Имя | Заметки |
|---|---|
| `exit-code` | Exit-код `indexnow submit`. |
| `submitted-count` | Сумма `urlCount` по всем батчам. `0` в `dry-run`. |
| `failed-count` | Батчи с не-2xx или ошибкой. `0` в `dry-run`. |
| `report` | Одной строкой; также пишется в `$GITHUB_STEP_SUMMARY`. |

## Рецепты

### На каждый push в `content/`

```yaml
name: indexnow
on:
  push:
    branches: [main]
    paths: ["content/**"]

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - uses: jtprogru/indexnow@v0
        with:
          key: ${{ secrets.INDEXNOW_KEY }}
          sitemap: https://example.com/sitemap.xml
          sitemap-since: ${{ github.event.before }}
          endpoint: bing,yandex
```

`sitemap-since: ${{ github.event.before }}` — строка, action прокидывает её в `--sitemap-since` (парсит RFC3339). Для push-event'ов это timestamp предыдущего HEAD: записи со старее `<lastmod>` пропускаются, остальное уходит.

### Каждый час по расписанию

```yaml
on:
  schedule:
    - cron: "0 * * * *"

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - uses: jtprogru/indexnow@v0
        with:
          key: ${{ secrets.INDEXNOW_KEY }}
          sitemap: https://example.com/sitemap.xml
```

Не хочется возиться с арифметикой дат в YAML — просто не передавайте `sitemap-since`, и каждый запуск пересубмитит весь sitemap. IndexNow идемпотентен, цена — HTTP-вызов.

### После Hugo / Eleventy сборки, явный список URL'ов

```yaml
- name: Build
  run: hugo --minify

- name: Collect changed URLs
  id: changed
  run: |
    {
      echo 'urls<<EOF'
      git diff --name-only "${{ github.event.before }}" HEAD -- 'content/**/*.md' \
        | sed 's#^content/\(.*\)\.md$#https://example.com/\1/#'
      echo EOF
    } >> "$GITHUB_OUTPUT"

- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    host: example.com
    urls: ${{ steps.changed.outputs.urls }}
```

Маппинг путь→URL — проект-специфичный (Hugo `permalinks`, Eleventy `permalink` в front-matter, …). Встроенный сабкоманд `from-diff` — в roadmap'е.

## Секреты

`INDEXNOW_KEY` кладите в repository или org-level secrets. Action берёт его как `input` и передаёт через env (`INDEXNOW_KEY`) — значение никогда не появляется в трасе command-line, даже с `set -x`.

Hosted key-файл: положите `<key>.txt` в корень static-сайта с содержимым, равным ключу. CLI-сабкоманд `verify` это проверяет; пока не выведен в input action'а — позовите `indexnow verify` следующим шагом, если нужно подтверждение в CI.

## Как устроена установка

На первом использовании в job:

1. Резолв версии (`inputs.version` → тег action'а → `latest`).
2. Скачивается `indexnow_{Linux|Darwin}_{x86_64|arm64}.tar.gz` и `checksums.txt`.
3. Sha256-проверка (`sha256sum -c` на Linux, `shasum -a 256` на macOS).
4. Распаковка в `$RUNNER_TOOL_CACHE/indexnow/<version>/<os-arch>/`.
5. Каталог кэшируется через `actions/cache` с ключом version+os+arch — последующие jobs (и reruns) пропускают скачивание.

`checksums.txt.sig` (GPG-подпись) — для out-of-band паранои; сам action полагается на TLS GitHub Releases + sha256.
