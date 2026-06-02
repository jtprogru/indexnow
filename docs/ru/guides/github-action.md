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
| `urls-from` | один из | — | Bash-сниппет, чей stdout трактуется как список URL'ов. См. [Кастомные источники URL](#kastomnye-istochniki-url-cherez-urls-from). |
| `sitemap-since` | нет | — | RFC3339; записи со старее `<lastmod>` отбрасываются. |
| `sitemap-timeout` | нет | дефолт CLI (`30s`) | Per-request HTTP timeout для sitemap. |
| `host` | нет | вывод из первого URL | Хост сайта (например `example.com`). |
| `key-location` | нет | derive из `host` + `key` | Абсолютный URL hosted-key файла. |
| `endpoint` | нет | `api` | Алиас или полный URL; comma-separated для fan-out. |
| `user-agent` | нет | `indexnow/<version>` | Заголовок `User-Agent`. |
| `config` | нет | — | Путь (относительно `$GITHUB_WORKSPACE`) к yaml-конфигу indexnow — defaults для `host`, `key`, `endpoint` и т.д. |
| `fail-on` | нет | `any` | `any\|4xx\|5xx\|never` — какие классы ответа дают exit ≠ 0. |
| `quiet` | нет | `false` | Глушит stdout CLI в логе шага. |
| `verbose` | нет | `false` | slog lifecycle/retry в stderr. |
| `dry-run` | нет | `false` | Печатает план и выходит, не идёт по сети. |
| `max-retries` | нет | дефолт CLI | Ретраи на 429/5xx/transport. |
| `base-backoff` | нет | дефолт CLI | Начальный backoff (Go duration, например `200ms`). |
| `max-backoff` | нет | дефолт CLI | Кап backoff'а. |
| `version` | нет | тег action'а | Какой релиз indexnow ставить. `"latest"` резолвится в рантайме. |
| `github-token` | нет | `${{ github.token }}` | Используется только для GitHub Releases API. |

Ровно один из `urls` / `file` / `sitemap` / `urls-from` должен быть задан — иначе preflight падает.

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

## Кастомные источники URL через `urls-from`

Когда ни `urls`, ни `file`, ни `sitemap` не подходят — соберите список сами. `urls-from` — bash-сниппет, чей stdout становится списком URL'ов (один URL на строку, `#` в начале — комментарий, пустые строки игнорируются). Запускается в `$GITHUB_WORKSPACE`, так что `git`, локально-выкаченные файлы и другие инструменты доступны сразу.

Пустой вывод — успех: step выходит с `submitted-count=0` и `submit` **не вызывается**. Non-zero exit вашего сниппета валит step (stderr пробрасывается в лог).

### Рецепт: только изменённые URL'ы из `git diff`

Исходный мотив. Маппинг путь→URL — проектно-специфичный (Hugo permalinks, Eleventy `permalink:` в front-matter, кастомные роутеры), поэтому он живёт в вашем сниппете, а не в схеме конфига, которую indexnow вынужден тащить годами.

```yaml
- uses: actions/checkout@v6
  with:
    fetch-depth: 0          # нужно, чтобы `git diff <base>..HEAD` резолвился
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    host: example.com
    urls-from: |
      git diff --name-only --diff-filter=AMR \
        "${{ github.event.before }}..${{ github.event.after }}" -- 'content/**/*.md' |
        sed 's#^content/\(.*\)\.md$#https://example.com/\1/#'
```

Краевой случай: force-push в новую ветку даёт `github.event.before = 0000000…` — оберните в `if:` и фоллбекните на `origin/main..HEAD`.

### Рецепт: sitemap с content-фильтром

Когда `--sitemap-since` недостаточно избирателен — например, нужны только URL'ы под определённым префиксом:

```yaml
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    urls-from: |
      curl -sSL https://example.com/sitemap.xml |
        grep -oE '<loc>[^<]+</loc>' |
        sed -E 's#</?loc>##g' |
        grep '^https://example.com/blog/'
```

### Рецепт: URL'ы из CMS API

```yaml
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    urls-from: |
      curl -sSL -H "Authorization: Bearer ${{ secrets.CMS_TOKEN }}" \
        https://cms.example.com/api/recently-published |
        jq -r '.items[].url'
```

`urls-from` — фича только Action'а. В CLI эквивалент — обычный shell-pipe в `indexnow submit --stdin`:

```bash
git diff --name-only HEAD~1..HEAD -- 'content/**/*.md' |
  sed 's#^content/\(.*\)\.md$#https://example.com/\1/#' |
  indexnow submit --stdin
```

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
