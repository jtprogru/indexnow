# Архитектура

```
indexnow/
├── action.yml               # определение composite GitHub Action (inputs/outputs/steps)
├── scripts/                 # шаговые скрипты action'а
│   ├── resolve-version.sh   # input → ref action'а → "latest"
│   ├── detect-platform.sh   # RUNNER_OS/ARCH → имена архивов goreleaser
│   ├── install.sh           # скачать + sha256-проверить + распаковать в tool cache
│   ├── preflight.sh         # проверка «ровно один источник URL»
│   └── run.sh               # сборка argv для submit, парс JSON, outputs + step summary
├── cmd/indexnow/main.go     # cobra-обвязка, ENV/config-defaults, exit-code-обвязка
└── internal/
    ├── cli/                 # логика в тестируемой форме (без cobra)
    │   ├── submit.go        # RunSubmit, сбор URL, fan-out по эндпоинтам, рендеринг, fail-on
    │   ├── keygen.go        # RunKeygen, запись hosted key-файла
    │   ├── verify.go        # RunVerify, проверка hosted-ключа
    │   └── errors.go
    ├── client/              # HTTP-клиент IndexNow
    │   ├── client.go        # Submit / SubmitBatch, retry-loop, marshal payload
    │   ├── endpoints.go     # резолв алиасов в URL (одиночный и comma-separated)
    │   ├── retry.go         # математика backoff / jitter
    │   └── errors.go
    ├── config/config.go     # опциональный yaml-конфиг (XDG-путь по умолчанию или --config)
    └── sitemap/sitemap.go   # потоковый парсер urlset / sitemapindex
```

## Разделение слоёв

- **`cmd/indexnow`** держит только cobra и OS-обвязку: флаги, сигналы, мердж flag > env > config > default, маппинг `cli.Exit*` в process exit code.
- **`internal/cli`** держит поведение CLI в тестируемом виде: `RunSubmit(ctx, opts, stdin, stdout, stderr, factory) int`, плюс `RunKeygen` и `RunVerify` в той же форме. Вход — `io.Reader`, выход — `io.Writer`, HTTP-клиент инжектится через `SubmitterFactory`, поэтому тестам не нужна сеть.
- **`internal/client`** держит wire-формат IndexNow и retry-политику. Безопасен для конкурентного использования; конфигурируется через `client.Config`.
- **`internal/config`** держит yaml-файл: строгий декод (неизвестные поля отвергаются), дефолтный путь `$XDG_CONFIG_HOME/indexnow/config.yaml` и различение «нет дефолтного файла — не ошибка, нет явного `--config` — ошибка».
- **`internal/sitemap`** держит wire-формат sitemap.org. Матчинг идёт по local name элемента, поэтому документы без `xmlns` тоже парсятся; поток идёт через `encoding/xml`, так что sitemap на 50 МБ / 50 000 записей не грузится в память целиком. Рекурсия по `<sitemapindex>` ограничена `MaxDepth = 5`, посещённые источники дедуплицируются — самоссылающиеся индексы завершаются корректно.
- **`action.yml` + `scripts/`** оборачивают релизный бинарь, а не переизобретают логику: action резолвит версию, сверяет checksum и вызывает `indexnow submit --output json`.

## Fan-out по эндпоинтам

`--endpoint` резолвится в список; каждый элемент получает свою goroutine и свой `client.Client`, батч URL'ов у всех общий. Результаты возвращаются как один `endpointBatch` на эндпоинт, в порядке входа — вывод детерминирован независимо от того, кто ответил первым. Wall-time равен времени самого медленного эндпоинта, а не сумме.

## Retry-политика

Клиент ретраит HTTP 429, 5xx и transport-ошибки. Backoff экспоненциальный с jitter'ом, ограничен `BaseBackoff` и `MaxBackoff`. Заголовок `Retry-After` парсится и как секунды, и как HTTP-date. Каждый `Result` несёт итоговый `StatusCode`, число `Attempts`, URL батча и итоговую ошибку (если есть).

## Батчинг

`SubmitBatch` режет вход на куски по `MaxBatchSize = 10000` (лимит протокола) и отдаёт по одному `Result` на HTTP-вызов, в порядке отправки.
