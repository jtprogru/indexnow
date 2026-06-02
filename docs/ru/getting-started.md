# Старт

Два способа запускать indexnow. Выбирайте под свой pipeline — движок и семантика протокола одни и те же.

## Получите IndexNow-ключ

Оба пути требуют ключ, размещённый по известному URL на сайте. Полный walkthrough — сгенерировать, выкатить, прописать, проверить, использовать, ротировать — в **[Жизненный цикл ключа](guides/key-lifecycle.md)**. Версия за 30 секунд:

```bash
brew install jtprogru/tap/indexnow   # или `go install`, см. ниже
indexnow key gen --write public/     # генерит ключ + создаёт public/<key>.txt
```

Закоммитьте `<key>.txt`, задеплойте, положите значение ключа в `INDEXNOW_KEY` секрет CI (или env / config-файл для локального использования) — и можно submit'ить.

## Путь A — GitHub Action (рекомендуется)

Если контент живёт в GitHub-репо, action — самый короткий путь:

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
```

Готово. `sitemap-since: ${{ github.event.before }}` значит «только URL'ы, у которых `<lastmod>` свежее предыдущего HEAD'а» — остальные пропускаются.

→ См. **[GitHub Action](guides/github-action.md)** — все inputs/outputs и ещё рецепты (по расписанию, после Hugo/Eleventy-сборки, явный список URL'ов).

## Путь B — CLI

CLI нужен, если indexnow запускается локально, по cron'у, или из CI, который не GitHub Actions.

### Установка

Homebrew (macOS / Linux):

```bash
brew tap jtprogru/tap
brew install indexnow
```

`go install`:

```bash
go install github.com/jtprogru/indexnow/cmd/indexnow@latest
```

Или готовый бинарь со страницы [Releases](https://github.com/jtprogru/indexnow/releases) (Linux / macOS / FreeBSD, amd64 / arm64).

### Первый вызов

```bash
export INDEXNOW_KEY=8f7e6d5c4b3a29180706050403020100
indexnow submit https://example.com/posts/new
```

Host выводится из URL'а, дефолтный эндпоинт — `https://api.indexnow.org/indexnow`. По спеке submission шарится со всеми поисковиками-участниками.

### Dry-run

```bash
indexnow submit --dry-run --file urls.txt
```

Покажет, что было бы отправлено, и выйдет с `0` — без сетевого вызова. Удобно для проверки sitemap'а перед боевым прогоном.

→ Полный референс команд: **[CLI-команды](commands/index.md)**.
