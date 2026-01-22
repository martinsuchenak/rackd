# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make
RUN apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community bun

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/build/rackd /usr/local/bin/rackd

RUN mkdir -p /data

ENV DATA_DIR=/data
ENV LISTEN_ADDR=:8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/api/datacenters || exit 1

ENTRYPOINT ["rackd"]
CMD ["server"]
