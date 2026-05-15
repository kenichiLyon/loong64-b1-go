# syntax=docker/dockerfile:1.7

ARG NODE_VERSION=24-bookworm-slim
ARG GO_VERSION=1.25-bookworm

FROM node:${NODE_VERSION} AS web-build
WORKDIR /src
COPY web/package*.json ./web/
RUN npm ci --prefix web
COPY web ./web
RUN npm run build --prefix web

FROM golang:${GO_VERSION} AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-build /src/web/dist ./web/dist
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -tags webui -ldflags="-s -w" -o /out/loong64-b1-go ./cmd/server

FROM scratch
COPY --from=go-build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-build /out/loong64-b1-go /loong64-b1-go
COPY --from=go-build /src/migrations /migrations
EXPOSE 8080
VOLUME ["/var/lib/loong64-b1-go"]
ENV HTTP_ADDR=0.0.0.0:8080 \
    APP_ENV=production \
    STORAGE_ROOT=/var/lib/loong64-b1-go/storage \
    RUNTIME_CONFIG_PATH=/var/lib/loong64-b1-go/config/runtime.json \
    UPGRADE_DIR=/migrations \
    DB_DRIVER=sqlite \
    SQLITE_PATH=/var/lib/loong64-b1-go/data/loong64-b1-go.db \
    AUTO_UPGRADE=true
ENTRYPOINT ["/loong64-b1-go"]
