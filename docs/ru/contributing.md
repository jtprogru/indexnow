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

## Релизы

Мейнтейнеры тегают с `main`:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

GoReleaser собирает бинари, подписывает checksum и обновляет Homebrew tap.
