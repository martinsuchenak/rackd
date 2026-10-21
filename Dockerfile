# Web UI stage
FROM oven/bun:1.4.3-alpine AS ui

WORKDIR /app/webui

COPY webui/package.json webui/bun.lock ./
RUN bun install --frozen-lockfile

COPY webui/ ./
RUN mkdir -p ../internal/ui && bun run build

# Build stage
FROM golang:1.27.3-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=ui /app/internal/ui/assets ./internal/ui/assets

RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /app/build/rackd .

# Runtime stage
FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/build/rackd /usr/local/bin/rackd

RUN mkdir -p /data

ENV DATA_DIR=/data
ENV LISTEN_ADDR=:8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/healthz || exit 1

ENTRYPOINT ["rackd"]
CMD ["server"]
