FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
	go build -trimpath -ldflags="-s -w" -o /out/help-desk ./cmd/server

FROM alpine:3.21 AS runtime

RUN adduser -D -H -u 10001 appuser && \
	apk add --no-cache ca-certificates wget

WORKDIR /app
COPY --from=builder /out/help-desk /app/help-desk

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
	CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/help-desk"]
