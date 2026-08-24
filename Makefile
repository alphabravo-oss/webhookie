.PHONY: build test frontend run docker-build docker-run tidy

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

docker-build:
	docker build -t webhookie:latest .

docker-run: docker-build
	docker run --rm -p 8080:8080 -e WEBHOOKIE_PUBLIC_BASE_URL=http://localhost:8080 -v webhookie-data:/data webhookie:latest
