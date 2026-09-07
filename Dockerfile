FROM migrate/migrate:v4.18.1 AS migrator

FROM golang:1.27-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
	go build -trimpath -ldflags="-s -w" -o /out/help-desk ./cmd/server

# Keep runtime free of apk (CDN often flaky); busybox wget is enough for healthcheck.
FROM alpine:3.21 AS runtime

RUN adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=migrator /usr/local/bin/migrate /usr/local/bin/migrate
COPY --from=builder /out/help-desk /app/help-desk
COPY migrations /migrations
COPY scripts/docker-entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && chown -R appuser:appuser /app /migrations

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=3 \
	CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/entrypoint.sh"]
