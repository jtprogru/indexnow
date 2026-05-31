# Архитектура

```
indexnow/
├── cmd/indexnow/main.go     # cobra-обвязка, ENV-defaults, exit-code-обвязка
└── internal/
    ├── cli/                 # логика в тестируемой форме (без cobra)
    │   ├── submit.go        # RunSubmit, сбор URL, рендеринг вывода, fail-on
    │   └── errors.go
    └── client/              # HTTP-клиент IndexNow
        ├── client.go        # Submit / SubmitBatch, retry-loop, marshal payload
        ├── endpoints.go     # резолв алиасов в URL
        ├── retry.go         # математика backoff / jitter
        └── errors.go
```

## Разделение слоёв

- **`cmd/indexnow`** держит только cobra и OS-обвязку: флаги, сигналы, ENV-fallback'и, маппинг `cli.Exit*` в process exit code.
- **`internal/cli`** держит поведение CLI в тестируемом виде: `RunSubmit(ctx, opts, stdin, stdout, stderr, factory) int`. Вход — `io.Reader`, выход — `io.Writer`, HTTP-клиент инжектится через `SubmitterFactory`, поэтому тестам не нужна сеть.
- **`internal/client`** держит wire-формат IndexNow и retry-политику. Безопасен для конкурентного использования; конфигурируется через `client.Config`.

## Retry-политика

Клиент ретраит HTTP 429, 5xx и transport-ошибки. Backoff экспоненциальный с jitter'ом, ограничен `BaseBackoff` и `MaxBackoff`. Заголовок `Retry-After` парсится и как секунды, и как HTTP-date. Каждый `Result` несёт итоговый `StatusCode`, число `Attempts`, URL батча и итоговую ошибку (если есть).

## Батчинг

`SubmitBatch` режет вход на куски по `MaxBatchSize = 10000` (лимит протокола) и отдаёт по одному `Result` на HTTP-вызов, в порядке отправки.
