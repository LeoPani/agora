.PHONY: help setup db-up db-down migrate \
        collect-openalex ingest-openalex \
        collect-locus ingest-locus \
        collect-inpi ingest-inpi \
        enrich-patents \
        collect-dgp ingest-dgp \
        collect-lens ingest-lens \
        collect-editais ingest-opportunities \
        collect-comex ingest-comex \
        collect-trends ingest-trends \
        collect-partners collect-linkedin ingest-partners \
        embed embed-publications embed-patents embed-server ingest-embeddings \
        collect-all ingest-all \
        run-api run-frontend run-scheduler build clean

INPUT ?= $(HOME)/Downloads/lens_export.csv

help: ## Mostra esta ajuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-26s\033[0m %s\n", $$1, $$2}'

# ── Setup ──────────────────────────────────────────────────────────────────────

setup: ## Setup deps Go + Python + Node
	cd backend && go mod download
	cd ai-service && python3 -m venv venv && \
		./venv/bin/pip install -r requirements.txt
	cd frontend && npm install

# ── Banco ──────────────────────────────────────────────────────────────────────

db-up: ## Sobe PostgreSQL com pgvector
	docker compose up -d postgres

db-down: ## Para PostgreSQL
	docker compose down

migrate: ## Roda todas as migrations
	cd backend && go run ./cmd/migrate up

migrate-status: ## Status das migrations
	cd backend && go run ./cmd/migrate status

# ── Coletores Python ───────────────────────────────────────────────────────────

collect-openalex: ## OpenAlex — publicações UFV
	cd ai-service && ./venv/bin/python3 collectors/openalex_collector.py

collect-locus: ## LOCUS — teses e dissertações UFV (DSpace 8)
	cd ai-service && ./venv/bin/python3 collectors/locus_collector.py

collect-inpi: ## INPI 775K — dataset HuggingFace
	cd ai-service && ./venv/bin/python3 collectors/inpi_dataset_loader.py

enrich-patents-py: ## Google Patents — enriquece patentes UFV (lento, ~235 req × 3s)
	cd ai-service && ./venv/bin/python3 collectors/google_patents_enricher.py

collect-dgp: ## DGP/CNPq — grupos de pesquisa UFV
	cd ai-service && ./venv/bin/python3 collectors/dgp_collector.py

collect-lens: ## Lens.org — processa CSV exportado manualmente (INPUT=caminho)
	cd ai-service && ./venv/bin/python3 collectors/lens_parser.py --input $(INPUT)

collect-editais: ## Editais — FAPEMIG + FINEP + CNPq + EMBRAPII
	cd ai-service && ./venv/bin/python3 collectors/fapemig_collector.py
	cd ai-service && ./venv/bin/python3 collectors/finep_collector.py
	cd ai-service && ./venv/bin/python3 collectors/cnpq_collector.py
	cd ai-service && ./venv/bin/python3 collectors/embrapii_collector.py

collect-comex: ## Comex Stat — gaps de importação (API MDIC)
	cd ai-service && ./venv/bin/python3 collectors/comex_collector.py

collect-trends: ## Google Trends — 32 keywords × 8 depts UFV
	cd ai-service && ./venv/bin/python3 collectors/trends_collector.py

embed: ## Gera embeddings semânticos (publicações + patentes)
	cd ai-service && ./venv/bin/python3 collectors/embedder.py

embed-publications: ## Gera embeddings só de publicações
	cd ai-service && ./venv/bin/python3 collectors/embedder.py --entity publications

embed-patents: ## Gera embeddings só de patentes
	cd ai-service && ./venv/bin/python3 collectors/embedder.py --entity patents

embed-server: ## Sobe servidor de embedding na porta 8082
	cd ai-service && ./venv/bin/python3 embed_server.py

ingest-embeddings: ## Ingere vetores semânticos no Postgres (pgvector)
	cd backend && go run ./cmd/collectors/ingest-embeddings

collect-partners: ## Interessados — empresas via CNPJ/Receita + pesquisadores via Lattes
	cd ai-service && ./venv/bin/python3 collectors/cnpj_partners_collector.py
	cd ai-service && ./venv/bin/python3 collectors/lattes_partners_collector.py

collect-linkedin: ## LinkedIn — gera queries de prospecção (sem scraping)
	cd ai-service && ./venv/bin/python3 collectors/linkedin_finder.py

collect-all: ## Roda TODOS os coletores em sequência (exceto Lens — manual)
	$(MAKE) collect-openalex
	$(MAKE) collect-locus
	$(MAKE) collect-inpi
	$(MAKE) collect-dgp
	$(MAKE) collect-editais
	$(MAKE) collect-comex
	$(MAKE) collect-trends

# ── Ingestores Go ──────────────────────────────────────────────────────────────

ingest-openalex: ## Ingere OpenAlex JSONL no Postgres
	cd backend && go run ./cmd/collectors/ingest-openalex

ingest-locus: ## Ingere LOCUS JSONL no Postgres
	cd backend && go run ./cmd/collectors/ingest-locus

ingest-inpi: ## Ingere INPI JSONL no Postgres (batch 5k)
	cd backend && go run ./cmd/collectors/ingest-inpi-dataset

enrich-patents: ## Atualiza patents com dados Google Patents
	cd backend && go run ./cmd/collectors/enrich-patents

ingest-dgp: ## Ingere DGP grupos no Postgres
	cd backend && go run ./cmd/collectors/ingest-dgp

ingest-lens: ## Ingere Lens.org JSONL no Postgres
	cd backend && go run ./cmd/collectors/ingest-lens

ingest-opportunities: ## Ingere editais_*.jsonl no Postgres
	cd backend && go run ./cmd/collectors/ingest-opportunities

ingest-comex: ## Ingere gaps de importação no Postgres
	cd backend && go run ./cmd/collectors/ingest-comex

ingest-trends: ## Ingere tendências de mercado no Postgres
	cd backend && go run ./cmd/collectors/ingest-trends

ingest-partners: ## Ingere parceiros/interessados no Postgres
	cd backend && go run ./cmd/collectors/ingest-partners

ingest-all: ## Roda TODOS os ingestores em sequência
	$(MAKE) ingest-openalex
	$(MAKE) ingest-locus
	$(MAKE) ingest-inpi
	$(MAKE) ingest-dgp
	$(MAKE) ingest-lens
	$(MAKE) ingest-opportunities
	$(MAKE) ingest-comex
	$(MAKE) ingest-trends
	$(MAKE) ingest-partners

# ── Dev ────────────────────────────────────────────────────────────────────────

run-api: ## Sobe API Go na porta 8081
	cd backend && go run ./cmd/api

run-frontend: ## Sobe frontend Next.js na porta 3000
	cd frontend && npm run dev

run-scheduler: ## Sobe scheduler de coleta automática
	cd backend && go run ./cmd/scheduler

# ── Build ──────────────────────────────────────────────────────────────────────

build: ## Compila todos os binários Go para backend/dist/
	mkdir -p backend/dist
	cd backend && \
		go build -o dist/agora-api           ./cmd/api && \
		go build -o dist/agora-migrate       ./cmd/migrate && \
		go build -o dist/agora-scheduler     ./cmd/scheduler && \
		go build -o dist/agora-ingest-openalex  ./cmd/collectors/ingest-openalex && \
		go build -o dist/agora-ingest-locus     ./cmd/collectors/ingest-locus && \
		go build -o dist/agora-ingest-inpi      ./cmd/collectors/ingest-inpi-dataset && \
		go build -o dist/agora-enrich-patents   ./cmd/collectors/enrich-patents && \
		go build -o dist/agora-ingest-dgp       ./cmd/collectors/ingest-dgp && \
		go build -o dist/agora-ingest-lens      ./cmd/collectors/ingest-lens && \
		go build -o dist/agora-ingest-opportunities ./cmd/collectors/ingest-opportunities && \
		go build -o dist/agora-ingest-comex     ./cmd/collectors/ingest-comex && \
		go build -o dist/agora-ingest-trends      ./cmd/collectors/ingest-trends && \
		go build -o dist/agora-ingest-embeddings ./cmd/collectors/ingest-embeddings

# ── AI Pipeline ────────────────────────────────────────────────────────────────

generate-embeddings: ## Gera embeddings de publicações, patentes e oportunidades
	cd ai-service && ./venv/bin/python3 embeddings/generate_embeddings.py

generate-embeddings-publications: ## Gera embeddings só de publicações
	cd ai-service && ./venv/bin/python3 embeddings/generate_embeddings.py --entity publications

generate-embeddings-patents: ## Gera embeddings só de patentes
	cd ai-service && ./venv/bin/python3 embeddings/generate_embeddings.py --entity patents

generate-embeddings-opportunities: ## Gera embeddings de oportunidades
	cd ai-service && ./venv/bin/python3 embeddings/generate_embeddings.py --entity opportunities

extract-editais: ## Re-extrai editais com LLM (requer GROQ_API_KEY)
	cd ai-service && ./venv/bin/python3 extractors/edital_extractor.py --all

eval-rag: ## Avaliação manual do Oráculo (RAG) — 10 perguntas de teste
	cd ai-service && ./venv/bin/python3 eval/rag_eval.py

# ── Ollama (opcional, custo zero) ──────────────────────────────────────────────

ollama-pull: ## Baixa modelos Ollama para uso local
	ollama pull llama3.1:8b
	ollama pull nomic-embed-text

ollama-serve: ## Sobe servidor Ollama local
	ollama serve

# ── Limpeza ────────────────────────────────────────────────────────────────────

clean: ## Remove artefatos de build e dados coletados
	rm -rf backend/dist
	rm -f ai-service/data/*.jsonl ai-service/data/*.json
