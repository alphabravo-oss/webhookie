.PHONY: build test frontend run docker-build docker-run docker-run-local tidy

frontend:
	cd frontend && npm ci && npm run build
	rm -rf internal/webui/dist
	cp -R frontend/dist internal/webui/dist

tidy:
	go mod tidy

build: frontend
	go build -o bin/webhookie ./cmd/webhookie

test:
	go test ./cmd/... ./internal/... ./examples/...
	cd frontend && npm test

run:
	go run ./cmd/webhookie

IMAGE ?= ghcr.io/alphabravo-oss/webhookie:0.1.0

docker-build:
	docker build --build-arg VERSION=0.1.0 -t webhookie:0.1.0 -t webhookie:latest .

docker-run:
	docker run --rm -p 8080:8080 -e WEBHOOKIE_PUBLIC_BASE_URL=http://localhost:8080 -v webhookie-data:/data $(IMAGE)

docker-run-local: docker-build
	$(MAKE) docker-run IMAGE=webhookie:0.1.0
