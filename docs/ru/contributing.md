# Контрибьютинг

PR'ы приветствуются. Проект маленький, церемоний нет.

## Локальная разработка

```bash
git clone https://github.com/jtprogru/indexnow
cd indexnow
make          # список доступных таргетов
make build    # бинарь в ./dist
make ci       # lint + race tests — то, что гоняет CI
```

CI зовёт те же таргеты (`make test-race`, `make docs-install`, `make docs-deploy`), а не дублирует команды инлайном — поэтому зелёный локальный `make ci` значит ровно то же, что зелёный workflow.

## Стиль

- `gofmt -s` (через `make fmt`).
- Конфиг `golangci-lint` — в `.golangci.yaml`. Запуск: `make lint`.
- Тесты в CI идут с `-race`; держите их зелёными под race-детектором.

## Сообщения коммитов

Conventional-ish, без жёсткого энфорсмента. Dependabot использует префиксы `chore(deps):` и `chore(ci):` — можно держать тот же стиль.

## Документация

Сайт — MkDocs Material, EN и RU собираются из `docs/en` и `docs/ru`. Языки держим синхронными: правка одного без второго — незавершённая правка.

```bash
make docs-install   # pip install -r docs/requirements.txt
make docs-serve     # http://127.0.0.1:8000
make docs-build     # mkdocs build --strict
```

Push в `main`, затрагивающий `docs/**` или `mkdocs.yml`, деплоит GitHub Pages.

## Релизы

Мейнтейнеры сначала фиксируют запись в CHANGELOG, потом тегают с `main` аннотированным GPG-подписанным тегом:

```bash
git tag -s vX.Y.Z -m "release vX.Y.Z"
git push origin vX.Y.Z
```

Push тега запускает два workflow:

- `goreleaser` — собирает бинари под Linux / macOS / FreeBSD (amd64 и arm64), GPG-подписывает `checksums.txt`, публикует GitHub Release и обновляет Homebrew cask в `jtprogru/homebrew-tap`.
- `release-major-alias` — force-двигает плавающие теги `vX` и `vX.Y` на новый релиз, чтобы `jtprogru/indexnow@v0` всегда указывал на свежий `v0.*`.

Если release-job упал на середине и вы его перезапускаете: goreleaser сконфигурирован с `release.replace_existing_artifacts: true` — без этого повторный прогон на уже опубликованном теге падает с `422 already_exists` на каждом ассете.
