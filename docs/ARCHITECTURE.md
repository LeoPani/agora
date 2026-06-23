# Ágora — Arquitetura

## Visão Geral

Ágora é um radar de inteligência de inovação que opera **antes da PI formal**.
Ingere dados de fontes públicas, gera embeddings vetoriais e produz sinais acionáveis
para NITs (Núcleos de Inovação Tecnológica) universitários.

## Stack

| Camada     | Tecnologia                              |
|------------|-----------------------------------------|
| Backend    | Go 1.24 (stdlib net/http, lib/pq)       |
| AI Service | Python 3.11+ (requests, sentence-BERT)  |
| Database   | PostgreSQL 16 + pgvector                |
| Frontend   | Next.js 16 + Tailwind v4 (JavaScript)   |

## Fluxo de dados

```
OpenAlex API → openalex_collector.py → JSONL → ingest-openalex → PostgreSQL
                                                                      ↓
                                                               embeddings (Fase 2)
                                                                      ↓
                                                            motor de sinais (Fase 3)
                                                                      ↓
                                                              API REST → Frontend
```

## Portas

- PostgreSQL: 5433 (para não conflitar com Argos em 5432)
- API Go: 8081 (para não conflitar com Argos em 8080)
- Frontend: 3000

## Pacotes Go internos

```
internal/
  config/         — configuração via env vars
  platform/
    database/     — pool PostgreSQL
    logger/       — slog estruturado
  domain/         — entidades puras (Researcher, Publication, ...)
  repository/
    postgres/     — queries SQL (upsert, find, link)
  api/            — handlers HTTP (Fase 2)
```
