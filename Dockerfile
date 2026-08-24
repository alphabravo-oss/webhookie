FROM node:22-alpine AS web
WORKDIR /web
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS go
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist /src/internal/webui/dist
RUN CGO_ENABLED=0 go build -o /webhookie ./cmd/webhookie

FROM alpine:3.20
ARG VERSION=0.1.0
RUN apk add --no-cache ca-certificates
RUN adduser -D -H -u 65532 nonroot
COPY --from=go /webhookie /webhookie
USER nonroot
EXPOSE 8080
VOLUME /data
ENV WEBHOOKIE_DATA_DIR=/data
ENV WEBHOOKIE_ADDR=:8080
ENV WEBHOOKIE_VERSION=${VERSION}
ENTRYPOINT ["/webhookie"]
