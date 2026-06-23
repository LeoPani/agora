# Fontes de Dados

## Camada 1 — Pesquisa (oferta da universidade)

| Fonte | Dado | Volume estimado | Status |
|-------|------|-----------------|--------|
| [OpenAlex API](https://openalex.org) | Publicações UFV com autores, abstracts, tópicos, citações, co-autorias | ~45.000 | **implementado** |
| [LOCUS REST API](https://locus.ufv.br) (DSpace 8) | Teses e dissertações completas | ~25.000 | planejado |
| DGP/CNPq | Grupos de pesquisa, linhas, pesquisadores | ~200 grupos | planejado |
| Embrapa | Pesquisa agropecuária relevante para UFV | ~10.000 | planejado |

### OpenAlex

- Gratuito, sem chave de API
- 100K requests/dia no polite pool (use `mailto` no parâmetro)
- UFV institution ID: `I4310312296`
- Endpoint: `GET https://api.openalex.org/works?filter=institutions.id:I4310312296`
- Coletor: `ai-service/collectors/openalex_collector.py`

---

## Camada 2 — PI (proteção existente)

| Fonte | Dado | Volume estimado | Status |
|-------|------|-----------------|--------|
| INPI Dataset (HuggingFace) | 775K patentes brasileiras com abstracts | 775.110 | planejado |
| Google Patents / Lens.org | Reivindicações, descrições, status legal, citações | ~500 UFV | planejado |
| Espacenet (OPS API) | Famílias internacionais de patentes | ~500 UFV | planejado |
| SNPC CultivarWeb | Cultivares protegidas | ~50 UFV | planejado |

---

## Camada 3 — Mercado (demandas externas)

| Fonte | Dado | Frequência | Status |
|-------|------|------------|--------|
| FAPEMIG, FINEP, CNPq, EMBRAPII | Editais abertos | Semanal | planejado |
| Comex Stat (MDIC) | Importações brasileiras por produto | Mensal | planejado |
| BNDES desembolsos | Capital direcionado por setor | Mensal | planejado |
| Google Trends | Termos em alta por área | Semanal | planejado |
| Patent trends (INPI 775K) | Áreas IPC em crescimento | Mensal | planejado |

---

## Camada 4 — Parceiros (potenciais contatos)

| Fonte | Dado | Tipo de parceiro | Status |
|-------|------|------------------|--------|
| INPI 775K | Empresas que patenteiam por área IPC | Industriais | planejado |
| Lens.org citations | Empresas que citam papers UFV em patentes | Interessados comprovados | planejado |
| tecnoPARQ / CenTev | Startups incubadas | Locais | planejado |
| OpenAlex co-autorias | Outras universidades | Acadêmicos | **disponível** (via Camada 1) |

---

## Notas de coleta

- **Idempotência**: todos os coletores usam `ON CONFLICT ... DO UPDATE` — podem rodar múltiplas vezes sem duplicar dados.
- **Rate limiting**: OpenAlex pede `time.sleep(0.1)` entre páginas no polite pool.
- **Dados brutos**: campo `raw_data JSONB` em cada tabela preserva o payload original para reprocessamento futuro.
- **Embeddings**: campo `embedding vector(384)` reservado — será preenchido na Fase 2 com Sentence-BERT português.
