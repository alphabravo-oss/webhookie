# Configuration

All settings are environment variables. There is no config file.

| Variable | Default | Meaning |
|---|---|---|
| `WEBHOOKIE_ADDR` | `:8080` | Listen address |
| `WEBHOOKIE_DATA_DIR` | `./data` | Directory for SQLite. Docker image sets `/data` |
| `WEBHOOKIE_PUBLIC_BASE_URL` | `http://localhost:8080` | Prefixed onto sink `url` fields and Copy URL in destination UIs |
| `WEBHOOKIE_MAX_BODY_BYTES` | `1048576` | Capture body cap (1 MiB). Oversize → `413` |
| `WEBHOOKIE_RETENTION_DAYS` | `7` | Prune events older than this |
| `WEBHOOKIE_MAX_EVENTS` | `10000` | Prune oldest when over this count |
| `WEBHOOKIE_PASSWORD` | empty | If set, UI + `/api/v1/*` require auth. **Never** `/hooks/*` |
| `WEBHOOKIE_VERSION` | `0.1.0` | Shown in UI footer and `/api/v1/meta`. Image builds set this from the git tag. |

Database path is always `$WEBHOOKIE_DATA_DIR/webhookie.db` (WAL). SQLite `MaxOpenConns=1`.

Prune runs about once a minute. Whichever of retention days or max events hits first wins.

## Auth (`WEBHOOKIE_PASSWORD`)

When set:

- Browser: HTTP Basic realm `webhookie`, username **`webhookie`**, password the env value. Successful login also sets cookie `webhookie_session`.
- API: Basic as above, or `?access_token=` equal to the password, or the session cookie.
- `POST /api/v1/login` with `{"password":"..."}` is allowed without prior auth.
- SPA static files are gated the same way as the UI.

Hooks stay unauthenticated on purpose: a password on `/hooks/slack/...` would break the product.

## CORS

API CORS allows `http://localhost:5173`, `http://127.0.0.1:5173`, and `WEBHOOKIE_PUBLIC_BASE_URL`. Needed for Vite dev.

## Docker

Image env defaults: `WEBHOOKIE_DATA_DIR=/data`, `WEBHOOKIE_ADDR=:8080`. Always set `WEBHOOKIE_PUBLIC_BASE_URL` to the URL your **apps** will call (often `http://localhost:8080` on a laptop, or `http://webhookie:8080` on a Compose network).
