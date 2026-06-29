# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-api                ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-migrate             ./cmd/migrate && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-scheduler           ./cmd/scheduler && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-ingest-openalex     ./cmd/collectors/ingest-openalex && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-ingest-locus        ./cmd/collectors/ingest-locus && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-ingest-inpi         ./cmd/collectors/ingest-inpi-dataset && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-ingest-dgp          ./cmd/collectors/ingest-dgp && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-ingest-lens         ./cmd/collectors/ingest-lens && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-ingest-opportunities ./cmd/collectors/ingest-opportunities && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-ingest-comex        ./cmd/collectors/ingest-comex && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-ingest-trends       ./cmd/collectors/ingest-trends && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-ingest-embeddings   ./cmd/collectors/ingest-embeddings && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/agora-enrich-patents      ./cmd/collectors/enrich-patents

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/ /usr/local/bin/

EXPOSE 8081

CMD ["agora-api"]
