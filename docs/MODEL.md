# Modelo de Matching

## Objetivo

Dado um par (entidade A, entidade B), o modelo estima a probabilidade de uma colaboração
produtiva: co-autoria, licenciamento, projeto cooperativo, pool de patentes.

---

## Ground Truth (positivos reais)

| Sinal | Fonte | Força |
|-------|-------|-------|
| Co-autoria OpenAlex | Dois pesquisadores publicaram juntos | Alta |
| Co-inventoria INPI | Depositaram patente juntos | Muito alta |
| Citação Lens.org | Empresa citou paper da UFV em patente | Alta (interesse comercial) |
| Licenciamento conhecido | Recombine↔UFV, Vina↔UFV, etc. | Máxima |

## Exemplos negativos

- Amostragem aleatória de pares na mesma área IPC sem interação histórica
- Pesquisadores com perfis semânticos muito distintos (controle)

---

## Métodos (em ordem de implementação)

### Fase 2 — Baseline

1. **TF-IDF + Cosine Similarity** — sobre títulos e abstracts; rápido, sem GPU
2. **Sentence-BERT português** (`neuralmind/bert-base-portuguese-cased` ou `paraphrase-multilingual-MiniLM-L12-v2`) — embeddings densos de 384 dimensões, indexados com ivfflat no pgvector

### Fase 3 — Radar funcional

3. **Grafo de co-autoria** — análise de buracos estruturais (Burt 1992): pares com competências complementares mas sem conexão direta são candidatos a match
4. **Filtragem colaborativa** — Netflix-style sobre matriz pesquisador × área IPC

### Fase 4 — Refinamento

5. **Cross-domain matching** — patente ↔ publicação (embedding do abstract comparado com embedding da reivindicação)
6. **Fine-tuning supervisionado** — quando ground truth suficiente (meta: ≥500 pares positivos confirmados pelo NIT)

---

## Tipos de Sinais Gerados

| # | Tipo | Lógica |
|---|------|--------|
| 1 | Pesquisa com potencial de PI | Área com crescimento IPC + papers sem patente correspondente + edital aberto |
| 2 | Match pesquisador ↔ empresa | Empresa citou paper da UFV em patente → sugerir aproximação |
| 3 | Gap de importação | Comex Stat importação alta na área + UFV tem patentes relevantes |
| 4 | Pool de patentes | UFV + outra instituição têm PI complementar na mesma cadeia tecnológica |
| 5 | Janela de oportunidade | Google Trends crescendo + BNDES investindo + UFV sem patentes na área |
| 6 | Colaboração interdepartamental | Dois pesquisadores com competências complementares sem publicação conjunta |

---

## Avaliação

- **Métrica principal**: Precision@10 (dos 10 matches sugeridos, quantos o NIT considera relevantes)
- **Baseline alvo**: >60% P@10 já com TF-IDF antes de embeddings
- **Feedback loop**: NIT marca matches como "relevante" / "irrelevante" → alimenta re-treinamento

---

## Schema relevante

```sql
-- Sinal gerado pelo motor
signals (signal_type, title, score, entities JSONB, reasoning JSONB, status)

-- Embeddings (Fase 2)
researchers.embedding  vector(384)
publications.embedding vector(384)
patents.embedding      vector(384)
opportunities.embedding vector(384)

-- Busca por similaridade (exemplo)
SELECT r.full_name, 1 - (r.embedding <=> $1::vector) AS similarity
FROM researchers r
ORDER BY r.embedding <=> $1::vector
LIMIT 10;
```
