# Commands

`indexnow` ships with one submit operation and a key namespace for everything else:

**Submission**

- [`indexnow submit`](submit.md) — submit one or more URLs to an IndexNow endpoint.

**Key management**

- [`indexnow key gen`](key-gen.md) — generate a new IndexNow key (optionally writing the hosted key file).
- [`indexnow key verify`](verify.md) — verify that the hosted key file matches the expected key.

The top-level `indexnow verify` form is kept as a backwards-compatibility alias for `indexnow key verify`. New code should prefer the canonical form.

Run `indexnow --help` or `indexnow <subcommand> --help` for the full flag set.
