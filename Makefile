.PHONY: help setup db-up db-down migrate \
        collect-openalex ingest-openalex \
        run-api run-frontend build clean

help: ## Mostra esta ajuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'

setup: ## Setup deps Go + Python + Node
	cd backend && go mod download
	cd ai-service && python3 -m venv venv && \
		./venv/bin/pip install -r requirements.txt
	cd frontend && npm install

db-up: ## Sobe PostgreSQL com pgvector
	docker compose up -d postgres

db-down: ## Para PostgreSQL
	docker compose down

migrate: ## Roda migrations
	cd backend && go run ./cmd/migrate up

collect-openalex: ## Coleta publicações UFV via OpenAlex
	cd ai-service && ./venv/bin/python3 collectors/openalex_collector.py

ingest-openalex: ## Ingere JSONL no Postgres
	cd backend && go run ./cmd/collectors/ingest-openalex

run-api: ## Sobe API Go
	cd backend && go run ./cmd/api

run-frontend: ## Soba frontend Next.js
	cd frontend && npm run dev

build: ## Compila binários Go
	cd backend && go build -o dist/agora-api ./cmd/api && \
		go build -o dist/agora-migrate ./cmd/migrate && \
		go build -o dist/agora-scheduler ./cmd/scheduler && \
		go build -o dist/agora-ingest-openalex ./cmd/collectors/ingest-openalex

clean: ## Remove artefatos de build
	rm -rf backend/dist
