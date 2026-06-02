# Жизненный цикл ключа

Протокол IndexNow использует hosted-file ключ как доказательство того, что вы владеете сайтом, для которого шлёте submission'ы. Эта страница проводит через весь цикл — сгенерировать, выкатить, прописать, проверить, использовать, ротировать — чтобы части складывались в общую картину.

Если нужен только быстрый напоминатель: `indexnow key gen --write public/`, закоммитить полученный `<key>.txt`, выкатить, дальше `indexnow submit ...`. Остальная страница — про «почему» и про углы, в которых стоит ориентироваться.

## Зачем ключ

IndexNow не требует регистрации/аккаунтов. Вместо этого каждый submission несёт:

- `key` — секретное значение.
- `keyLocation` — URL на вашем сайте, по которому это значение и отдаётся.

Поисковики идут на `keyLocation` по HTTPS, читают тело, тримят whitespace, сравнивают. Совпало → submission принят как пришедший от того, кто контролирует домен. Несовпало / 404 → тихо отброшен.

Так что ключ это, по сути, «токен, который вы кладёте на свой сайт чтобы доказать, что вы можете класть файлы на свой сайт». Генерируется один раз, используется много.

## 1. Bootstrap — сгенерировать ключ

```bash
indexnow key gen --write public/
```

Что происходит:

- 32 случайных hex-символа из `crypto/rand` (128 бит энтропии).
- Создаётся файл `public/<key>.txt` с правами `0644` и одной строкой содержимого — самим ключом.
- Ключ печатается в stdout; `wrote public/<key>.txt` идёт в stderr.

Pipe-friendly:

```bash
KEY=$(indexnow key gen --write public/)
```

Директория `public/` должна **уже существовать** — `indexnow` не делает `mkdir -p`. `--force` нужен, если переписать существующий файл с тем же именем (редкий случай — например, при тестах с seeded-энтропией).

## 2. Deploy — сделать файл live

Закоммитить `<key>.txt` в репо и запушить. После сборки и деплоя сайта файл должен быть доступен по URL'у, который IndexNow ожидает (следующая секция).

Три условия для IndexNow:

- HTTPS (`https://`, не `http://`).
- HTTP 200 OK.
- Тело ответа, триммированное по краям, точно равно ключу.

Большинство SSG обрабатывают `public/`, `static/`, `content/` как pass-through-ассеты — киньте файл в нужную директорию своего генератора, этого достаточно.

## 3. Два режима hosting'а

Существуют две конвенции, где может жить файл. Выберите одну и придерживайтесь её.

### Default: `https://<host>/<key>.txt`

Если нигде не передан `--key-location`, и indexnow, и поисковики предполагают, что файл лежит в корне домена. Самый простой путь, без лишней настройки.

```bash
# bootstrap
indexnow key gen --write public/

# use
indexnow submit https://example.com/posts/foo --host example.com --key $KEY
```

Поисковики идут на `https://example.com/<key>.txt`.

### Explicit `keyLocation`

Файл можно положить где угодно на домене — удобно, когда в корне «тесно» и вы держите все верификационные файлы под одним префиксом:

```bash
indexnow key gen --write public/.well-known/indexnow/
indexnow submit https://example.com/posts/foo \
  --host example.com --key $KEY \
  --key-location https://example.com/.well-known/indexnow/$KEY.txt
```

Trade-off: каждый submission должен нести явный `--key-location` (или иметь его в env / config). В default-режиме это деривится автоматически.

Решите один раз, задокументируйте в `.indexnow.yaml` репо — и забудьте.

## 4. Wire it up — где ключ живёт в момент использования

Ключ появляется в трёх ортогональных местах. Каждое — для своей аудитории.

| Где | Аудитория | Механизм |
|---|---|---|
| Env `INDEXNOW_KEY` | Локальный shell, ad-hoc CLI | `export INDEXNOW_KEY=...` |
| `~/.config/indexnow/config.yaml` | Машина разработчика, persistent | YAML-поле `key:` |
| `.indexnow.yaml` в репо | Проектные defaults (host, endpoint) | YAML; **`key:` в публичном репо НЕ коммитьте** |
| GitHub Actions secret | CI-workflow | `${{ secrets.INDEXNOW_KEY }}` → action input `key:` |

Precedence (более конкретное побеждает): CLI-флаг > environment > config-файл > built-in default.

Типовая CI-настройка:

1. Положите ключ в repository-секрет `INDEXNOW_KEY`.
2. Не-секретные defaults (`host`, `endpoint`, `user_agent`) — в коммитнутый `.indexnow.yaml`.
3. Передавайте секрет в input action'а; action прокинет его в `INDEXNOW_KEY` env'у внутри.

## 5. Verify — убедиться, что bootstrap прошёл

После первого деплоя — запустите один раз:

```bash
indexnow key verify --host example.com --key $KEY
```

Exit `0` — hosted-файл скачался корректно и совпал с ключом. Exit `1` — несовпадение, не-200 или сетевая ошибка; поисковики молча отбросят все submission'ы, пока вы не почините hosting. `--verbose` показывает полный URL и HTTP-статус.

`verify` не нужен на каждый push — это one-shot bootstrap-проверка. Перезапустите, если: в IndexNow-ответах внезапно появились 4xx, после крупных infra-изменений, или когда коллега подозревает, что hosting дрейфанул.

## 6. Use — отправить URL'ы

Любой путь:

```bash
# CLI
indexnow submit https://example.com/posts/foo

# GitHub Action
- uses: jtprogru/indexnow@v0
  with:
    key: ${{ secrets.INDEXNOW_KEY }}
    sitemap: https://example.com/sitemap.xml
```

Подробнее — [CLI-команды → submit](../commands/submit.md) и [GitHub Action](github-action.md).

## 7. Ротация (пока вручную)

`indexnow key rotate` в roadmap'е. Пока он не вышел — ротируйте руками, когда нужно (большинству проектов это никогда не нужно):

1. Сгенерить новый ключ рядом со старым: `indexnow key gen --write public/`. Теперь и `<old>.txt`, и `<new>.txt` — оба live.
2. Обновить место, где хранится ключ (env / config / GitHub secret) на новое значение. Submission'ы сразу начинают нести новый ключ.
3. Ждать. Поисковики агрессивно кешируют `keyLocation`; дайте 1–2 недели grace-периода, чтобы in-flight верификации против старого ключа отрезолвились.
4. Удалить `<old>.txt` из репо и задеплоить.

Grace-период — awkward-часть; именно поэтому `key rotate` это v0.8-задача, которую стоит сделать правильно, а не как однострочник в shell.

## Типовые грабли

- **Имя файла ≠ ключ.** Если делаете файл руками без `--write`, проверьте, что имя (без `.txt`) точно равно ключу. Несовпадение — самая частая причина failed verify.
- **Trailing newline.** `indexnow key gen --write` добавляет один. Все известные IndexNow-эндпоинты тримят whitespace перед сравнением — leading/trailing `\n` ок при ручном создании файла; embedded whitespace и BOM'ы — нет.
- **HTTP вместо HTTPS.** Редирект с `http://` на `https://` работает для браузеров, но некоторые IndexNow-верификаторы не следуют за ним. Хост'те файл сразу по canonical HTTPS-URL'у.
- **Robots.txt или auth перед файлом.** Endpoint должен отдавать 200 анонимным клиентам без rate-limit'ов.
- **`--write` в несуществующую директорию.** `indexnow` отказывается осознанно. Создайте директорию заранее.
