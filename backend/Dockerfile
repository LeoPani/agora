# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-api       ./cmd/api      && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-migrate    ./cmd/migrate  && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-scheduler  ./cmd/scheduler

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/agora-api       /usr/local/bin/agora-api
COPY --from=builder /out/agora-migrate   /usr/local/bin/agora-migrate
COPY --from=builder /out/agora-scheduler /usr/local/bin/agora-scheduler

EXPOSE 8081

CMD ["agora-api"]
