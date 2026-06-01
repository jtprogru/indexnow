# Endpoints

IndexNow defines a shared submission model: a POST to any one participating endpoint is replicated to every other participant. You only need to call one.

`--endpoint` accepts a short alias or a full URL:

| Alias    | URL                                            |
|----------|------------------------------------------------|
| `api`    | `https://api.indexnow.org/indexnow` (default)  |
| `bing`   | `https://www.bing.com/indexnow`                |
| `yandex` | `https://yandex.com/indexnow`                  |
| `naver`  | `https://searchadvisor.naver.com/indexnow`     |
| `seznam` | `https://search.seznam.cz/indexnow`            |
| `yep`    | `https://indexnow.yep.com/indexnow`            |

Anything starting with `http://` or `https://` is passed through verbatim, which is useful for staging endpoints and future participants.

## Multiple endpoints

`--endpoint` accepts a comma-separated list and submits to every endpoint in parallel:

```
indexnow submit --endpoint bing,yandex https://example.com/post/1
```

Aliases and full URLs can be mixed; duplicates are removed while order is preserved. Each endpoint runs in its own goroutine so total wall-time tracks the slowest single endpoint, not the sum.

The protocol guarantees that one submission propagates to all participants, so multi-endpoint is for explicit redundancy — for example, when you want independent acknowledgements or are debugging which participant is slow.

### Output

- **text**: with one endpoint the output is unchanged. With multiple endpoints each line is prefixed with `endpoint=<url>` so the source is unambiguous.
- **json**: the array gains one entry per endpoint × batch combination, each tagged with its `endpoint` field.

### Exit code with multiple endpoints

`--fail-on` is evaluated across all endpoints — if any endpoint's batches trip the policy, the process exits non-zero. Endpoint-level errors (factory init, transport failure that aborts the whole batch) always produce a non-zero exit, regardless of `--fail-on`.
