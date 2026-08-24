# Getting started

> **Alpha.** Use at your own risk.

Webhookie is a single Go process. In Docker the UI is embedded. In local UI development, Vite and Go run separately.

## Docker (recommended)

Public image on GHCR (linux/amd64, linux/arm64):

```bash
docker pull ghcr.io/alphabravo-oss/webhookie:0.1.0
docker run --rm -p 8080:8080 \
  -e WEBHOOKIE_PUBLIC_BASE_URL=http://localhost:8080 \
  -v webhookie-data:/data \
  ghcr.io/alphabravo-oss/webhookie:0.1.0
```

Tags: `0.1.0` (this release), `0.1`, `latest`. Pulls are anonymous once the package is public. If `docker pull` returns `denied`, an org owner must set the GHCR package visibility to public (Package settings on `ghcr.io/alphabravo-oss/webhookie`).

Compose (from repo root; uses the GHCR image):

```bash
docker compose -f deploy/docker-compose.yml up
```

`deploy/docker-compose.yml` also has a `build:` context if you pass `--build`. It publishes `8080:8080`, sets `WEBHOOKIE_PUBLIC_BASE_URL=http://localhost:8080`, and mounts volume `webhookie-data` at `/data`.

Open http://localhost:8080. SQLite is `$WEBHOOKIE_DATA_DIR/webhookie.db` (`/data/webhookie.db` in the image).

### Build the image yourself

The `Dockerfile` is multi-stage:

1. `node:22-alpine` — `npm ci` + `npm run build` in `frontend/`
2. `golang:1.26-alpine` — `CGO_ENABLED=0` build of `./cmd/webhookie`, copies `frontend/dist` into `internal/webui/dist`
3. `alpine:3.20` — binary as user `nonroot` (uid 65532), `EXPOSE 8080`, `VOLUME /data`, `WEBHOOKIE_VERSION` from build-arg

```bash
make docker-build       # tags webhookie:0.1.0 and webhookie:latest
make docker-run-local   # run that local tag
docker build --build-arg VERSION=0.1.0 -t webhookie:0.1.0 .
```

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

Provider URLs, validation, and follow-ups (`response_url`, Discord message PATCH, `answerCallbackQuery`): [sinks.md](sinks.md). Destination click theater: [destinations.md](destinations.md).

## Health

```bash
curl -s http://localhost:8080/healthz   # {"status":"ok"}
curl -s http://localhost:8080/readyz    # SQLite ping
curl -s http://localhost:8080/metrics   # Prometheus text
```
