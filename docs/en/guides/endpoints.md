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
