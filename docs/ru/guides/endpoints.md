# Эндпоинты

IndexNow устроен как общая шина submission'ов: POST на любого участника автоматически реплицируется ко всем остальным. Достаточно дёрнуть один.

`--endpoint` принимает алиас или полный URL:

| Алиас    | URL                                            |
|----------|------------------------------------------------|
| `api`    | `https://api.indexnow.org/indexnow` (default)  |
| `bing`   | `https://www.bing.com/indexnow`                |
| `yandex` | `https://yandex.com/indexnow`                  |
| `naver`  | `https://searchadvisor.naver.com/indexnow`     |
| `seznam` | `https://search.seznam.cz/indexnow`            |
| `yep`    | `https://indexnow.yep.com/indexnow`            |

Всё, что начинается с `http://` или `https://`, прокидывается как есть — удобно для staging-эндпоинтов и будущих участников.
