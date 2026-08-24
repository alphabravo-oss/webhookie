# Getting started

> **Alpha.** Use at your own risk.

Webhookie is a single Go process. In Docker the UI is embedded. In local UI development, Vite and Go run separately.

## Docker (recommended)

The `Dockerfile` is multi-stage:

1. `node:22-alpine` — `npm ci` + `npm run build` in `frontend/`
2. `golang:1.26-alpine` — `CGO_ENABLED=0` build of `./cmd/webhookie`, copies `frontend/dist` into `internal/webui/dist`
3. `alpine:3.20` — binary as user `nonroot` (uid 65532), `EXPOSE 8080`, `VOLUME /data`

```bash
docker build -t webhookie:latest .
docker run --rm -p 8080:8080 \
  -e WEBHOOKIE_PUBLIC_BASE_URL=http://localhost:8080 \
  -v webhookie-data:/data \
  webhookie:latest
```

Open http://localhost:8080. SQLite is `$WEBHOOKIE_DATA_DIR/webhookie.db` (`/data/webhookie.db` in the image).

Makefile:

```bash
make docker-build   # docker build -t webhookie:latest .
make docker-run     # build + run with the named volume webhookie-data
```

Compose (from repo root):

```bash
docker compose -f deploy/docker-compose.yml up --build
```

`deploy/docker-compose.yml` builds context `..` (repo root), publishes `8080:8080`, sets `WEBHOOKIE_PUBLIC_BASE_URL=http://localhost:8080`, and mounts volume `webhookie-data` at `/data`.

There is no assumed published image. README’s `ghcr.io/...` line was removed; build from this tree.

## Binary without Docker

Needs Node 22+ (frontend build) and Go 1.25+ (module is `go 1.25.0`, toolchain `go1.26.2`).

```bash
make build    # npm ci, vite build, copy dist, go build → bin/webhookie
./bin/webhookie
```

Or:

```bash
make frontend
go run ./cmd/webhookie
```

## Local UI development

Two processes. Vite HMR on port 5173; API on port 8080. Vite proxies `/api` and `/hooks` to the API (see `frontend/vite.config.ts`). Hit `/healthz` on port 8080 directly.

```bash
# terminal 1
WEBHOOKIE_PUBLIC_BASE_URL=http://localhost:8080 go run ./cmd/webhookie

# terminal 2
cd frontend && npm install && npm run dev
```

Use http://localhost:5173 in the browser. Destination “Copy URL” still uses `WEBHOOKIE_PUBLIC_BASE_URL` (the API), so apps should POST to `:8080`.

## Tests

```bash
make test
# go test ./cmd/... ./internal/... ./examples/...
# cd frontend && npm test
```

`e2e/` is empty. There is no Playwright suite yet.

## Health

```bash
curl -s http://localhost:8080/healthz   # {"status":"ok"}
curl -s http://localhost:8080/readyz    # SQLite ping
curl -s http://localhost:8080/metrics   # Prometheus text
```
