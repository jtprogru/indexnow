# Контрибьютинг

PR'ы приветствуются. Проект маленький, церемоний нет.

## Локальная разработка

```bash
git clone https://github.com/jtprogru/indexnow
cd indexnow
task          # список доступных задач
task build    # бинарь в ./dist
task ci       # lint + race tests — то, что гоняет CI
```

## Стиль

- `gofmt -s` (через `task fmt`).
- Конфиг `golangci-lint` — в `.golangci.yaml`. Запуск: `task lint`.
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
