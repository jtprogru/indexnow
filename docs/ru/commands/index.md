# Команды

В `indexnow` одна submit-операция и namespace для работы с ключами:

**Отправка**

- [`indexnow submit`](submit.md) — отправить один или несколько URL на IndexNow-эндпоинт.

**Управление ключами**

- [`indexnow key gen`](key-gen.md) — сгенерировать новый IndexNow-ключ (опционально с hosted key-файлом).
- [`indexnow key verify`](verify.md) — проверить, что hosted key-файл совпадает с ожидаемым ключом.

Топ-левел форма `indexnow verify` сохранена как backwards-compat alias для `indexnow key verify`. В новом коде используйте canonical-форму.

Полный набор флагов — `indexnow --help` / `indexnow <subcommand> --help`.
