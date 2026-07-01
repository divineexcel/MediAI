# ─── Stage 1: Build ──────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags="-w -s" -o medisave ./cmd/server

# ─── Stage 2: Run ────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache sqlite ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Africa/Lagos /etc/localtime && \
    echo "Africa/Lagos" > /etc/timezone && \
    addgroup -S medisave && adduser -S medisave -G medisave

WORKDIR /app

COPY --from=builder /app/medisave .
COPY --from=builder /app/web ./web

RUN mkdir -p /app/data && chown -R medisave:medisave /app

USER medisave

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
  CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["./medisave"]
