# Ágora by Argos

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Next JS](https://img.shields.io/badge/Next-black?style=for-the-badge&logo=next.js&logoColor=white)
![Postgres](https://img.shields.io/badge/postgres-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)
![pgvector](https://img.shields.io/badge/pgvector-purple?style=for-the-badge)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)
[![Dataset](https://img.shields.io/badge/Dataset-INPI_775K-blue?style=for-the-badge&logo=huggingface)](https://huggingface.co/datasets/LeoPani/inpi-patent-abstracts)

**Radar de Inteligência de Inovação para NITs Brasileiros**

![Dashboard](docs/screenshot-dashboard.png)

Ágora é um sistema que escaneia automaticamente o ecossistema de inovação universitário e gera sinais acionáveis para Núcleos de Inovação Tecnológica. Onde o Argos gerencia a propriedade intelectual existente, o Ágora identifica o que pode virar PI, quem deve colaborar com quem, e onde estão as oportunidades de transferência de tecnologia.

Projeto piloto com o NIT.UFV (Universidade Federal de Viçosa).

---

## Quick Start

Para rodar o projeto localmente com as configurações padronizadas de testes:

```bash
# 1. Instalar dependências (Go, Node, Python venv)
make setup

# 2. Subir banco de dados e aplicar migrations
make db-up
make migrate

# 3. Coletar e Ingerir dados (ex: OpenAlex)
make collect-openalex
make ingest-openalex

# 4. Iniciar servidor de Embeddings (em outro terminal)
make run-embed-server

# 5. Gerar vetores semânticos no banco
make generate-embeddings

# 6. Rodar os serviços da aplicação
make run-api       # Terminal 3
make run-frontend  # Terminal 4
```

---

## Visão

O Ágora atua antes da PI formal. Ele responde três perguntas:

1. **O que a universidade tem?** Publicações, teses, projetos, competências dos pesquisadores, PI existente.
2. **O que o mundo precisa?** Editais, empresas buscando tecnologia, tendências de mercado, gaps de importação, setores em crescimento.
3. **Quem conectar com quem?** Cruzamento com IA para sugerir parcerias, licenciamentos, depósitos de PI, pools de patentes.

A PI é uma saída possível, não o ponto de partida.

## Por que existe

O Sistema Financiar (FUNARBE/UFV) foi descontinuado em agosto de 2024 após 21 anos de operação. Era a principal ferramenta de prospecção de editais para NITs brasileiros. Existe um vácuo que precisa ser preenchido, e a tecnologia disponível hoje (IA, embeddings semânticos, scraping robusto, dados abertos) permite algo muito mais sofisticado do que o Financiar entregava.

---

## Architecture

```text
┌──────────────────────────────────────────────────────────────┐
│                        FRONTEND (Next.js)                    │
│   Dashboard | /signals | /opportunities | /oraculo | /agente │
└──────────────────────────────┬───────────────────────────────┘
                               │ HTTP / REST
┌──────────────────────────────▼───────────────────────────────┐
│                          BACKEND (Go)                        │
│                                                              │
│  ┌────────────────┐     ┌──────────────┐     ┌────────────┐  │
│  │ API Endpoints  │◄───►│ RAG Retriever│◄───►│ ReAct Agent│  │
│  └────────────────┘     └───────┬──────┘     └──────┬─────┘  │
│          ▲                      │                   │        │
│          │                      ▼                   ▼        │
│  ┌───────▼────────┐     ┌──────────────┐     ┌────────────┐  │
│  │ Internal LLM   │◄───►│   AI Router  │◄───►│  Providers │  │
│  │ Module (cost)  │     │(Groq,Gemini..)     │            │  │
│  └────────────────┘     └──────────────┘     └────────────┘  │
└──────────────────────────────┬───────────────────────────────┘
                               │
            ┌──────────────────┼────────────────────┐
            │                  │                    │
┌───────────▼────────┐ ┌───────▼────────┐ ┌─────────▼──────────┐
│   PostgreSQL DB    │ │ PYTHON WORKERS │ │ EMBEDDING SERVER   │
│   (com pgvector)   │ │ (ai-service)   │ │ port: 8082         │
└───────────┬────────┘ └───────┬────────┘ └─────────┬──────────┘
            │                  │                    │
            └──────────────────┴────────────────────┘
                               │
   ┌───────────────────┬───────┴───────┬───────────────────┐
   ▼                   ▼               ▼                   ▼
Camada 1:          Camada 2:       Camada 3:           Camada 4:
PESQUISA           PI              MERCADO             PARCEIROS
- OpenAlex         - INPI 775K     - Editais (4x)      - Empresas
- Locus DSpace     - Google Pat.   - Comex Stat        - Citações
- Grupos CNPq      - Lens.org      - Google Trends     - Lattes
```

### Camada 1: Pesquisa (oferta da universidade)
| Fonte | Dado | Volume estimado |
|-------|------|-----------------|
| OpenAlex API | Publicações UFV com autores, abstracts, tópicos, citações, co-autorias | ~45.000 |
| LOCUS REST API (DSpace 8) | Teses e dissertações completas | ~25.000 |
| DGP/CNPq | Grupos de pesquisa, linhas, pesquisadores | ~200 grupos |
| Embrapa (relevante para UFV) | Pesquisa agropecuária | ~10.000 |

### Camada 2: PI (proteção existente)
| Fonte | Dado | Volume estimado |
|-------|------|-----------------|
| INPI Dataset (HuggingFace) | 775K patentes brasileiras com abstracts | 775.110 |
| Google Patents | Reivindicações e descrições completas | ~500 UFV |
| Espacenet (OPS API) | Famílias internacionais de patentes | ~500 UFV |
| Lens.org | Patentes + status legal + citações a papers | ~500 UFV |
| SNPC CultivarWeb | Cultivares protegidas | ~50 UFV |
| RPI (INPI) | Software, marcas, atualizações semanais | ~20 UFV |

### Camada 3: Mercado (demandas externas)
| Fonte | Dado | Frequência de atualização |
|-------|------|---------------------------|
| FAPEMIG, FINEP, CNPq, EMBRAPII | Editais abertos | Semanal |
| Comex Stat (MDIC) | Importações brasileiras por produto | Mensal |
| RAIS/CAGED | Empregos por setor | Trimestral |
| BNDES desembolsos | Capital direcionado por setor | Mensal |
| Google Trends | Termos em alta por área | Semanal |
| Patent trends (INPI 775K) | Áreas IPC em crescimento | Mensal |

### Camada 4: Parceiros (potenciais contatos)
| Fonte | Dado | Tipo de parceiro |
|-------|------|------------------|
| INPI 775K | Empresas que patenteiam por área | Industriais |
| Lens.org citations | Empresas que citam papers UFV | Interessados |
| tecnoPARQ/CenTev | Startups incubadas | Locais |
| EMBRAPII unidades | Empresas em projetos cooperativos | Validados |
| OpenAlex co-autorias | Outras universidades | Acadêmicos |
| Comex Stat importadores | Empresas que importam tecnologia | Potenciais |

## Modelo de Matching

### Ground truth (positivos reais)
- Co-autorias do OpenAlex: dois pesquisadores publicaram juntos = match validado
- Co-inventorias do INPI: depositaram patente juntos = match forte
- Citações Lens.org: empresa citou paper da UFV em patente = interesse comercial
- Licenciamentos conhecidos (Recombine ↔ UFV, Vina ↔ UFV, etc.)

### Exemplos negativos
- Amostragem aleatória de pares na mesma área IPC que nunca interagiram
- Pesquisadores com perfis muito distintos (controle)

### Métodos combinados
1. TF-IDF + Cosine similarity (baseline rápido)
2. Sentence-BERT em português (embeddings semânticos densos)
3. Grafo de co-autoria com análise de buracos estruturais
4. Filtragem colaborativa (Netflix-style)
5. Cross-domain matching (patente ↔ publicação)
6. Fine-tuning supervisionado quando tivermos ground truth suficiente

## Tipos de Sinais Gerados pelo Radar

**Sinal 1 — Pesquisa com potencial de PI**
"O DFP publicou 8 papers sobre biocontrole com nanopartículas em 2025.
Nenhuma patente depositada. Área C12 cresceu 34% no INPI. 3 empresas
patentearam nessa área. Edital FAPEMIG aberto. → Oportunidade Alta."

**Sinal 2 — Match pesquisador ↔ empresa**
"A Bayer citou 2 papers do Prof. X em patentes de defensivos biológicos.
Nunca houve contato formal. → Sugerir aproximação."

**Sinal 3 — Gap de importação**
"Brasil importou R$ 200M em bioinsumos em 2025. UFV tem 15 patentes em
bioinsumos. → Potencial de substituição de importação."

**Sinal 4 — Pool de patentes**
"UFV tem patente em formulação (DQI), UFLA tem aplicação (DEA), Embrapa
tem cultivar resistente. → Sugerir pool RMPI."

**Sinal 5 — Janela de oportunidade**
"Google Trends: 'agricultura regenerativa' +200% em 2 anos. BNDES
desembolsou R$ 500M no setor. UFV tem 3 grupos de pesquisa na área
mas 0 patentes. → Janela aberta."

**Sinal 6 — Colaboração interdepartamental**
"Profª A (DFP) e Prof. B (DQI) têm competências complementares mas
nunca publicaram juntos. → Sugerir projeto conjunto."

## Stack Técnica

- **Backend**: Go (workers de scraping, API REST)
- **Workers Python**: para tarefas de NLP (embeddings, classificação)
- **Banco**: PostgreSQL com pgvector (busca por similaridade vetorial)
- **Frontend**: Next.js + Tailwind (mesma estética do Argos, branding
  "Ágora by Argos")
- **Modelos**: Sentence-BERT português + classificadores customizados
- **Deploy**: Vercel (frontend) + Railway/Render (backend)

## Roadmap

### Fase 1 — Data lake (semanas 1-3)
- Coletor OpenAlex (45K publicações UFV + co-autorias)
- Coletor LOCUS expandido (25K teses/dissertações)
- Coletor Lens.org (citações patente↔scholar)
- Integração com dataset INPI 775K existente
- PostgreSQL com pgvector configurado
- Workers automáticos com cron

### Fase 2 — Modelo base (semanas 4-5)
- Geração de embeddings para todas as publicações e patentes
- Construção do grafo de co-autoria
- Baseline TF-IDF + Sentence-BERT
- Avaliação contra ground truth (co-autorias conhecidas)

### Fase 3 — Radar funcional (semanas 6-7)
- Scrapers de editais (FAPEMIG, FINEP, CNPq, EMBRAPII)
- Integração Comex Stat, Google Trends, BNDES
- Motor de geração de sinais
- Frontend Ágora by Argos com 6 tipos de sinais

### Fase 4 — Refinamento e fine-tuning (semanas 8+)
- Fine-tuning supervisionado com ground truth expandido
- Active learning com feedback do NIT
- Pool de patentes e matching multi-institucional (RMPI)

## Princípios

1. **Dados públicos primeiro**: tudo que dá pra coletar sem autorização
   é coletado agora. Acesso a sistemas internos da UFV (SISPPG, NITSys)
   vem depois, se a parceria evoluir.

2. **Raspagem robusta**: scrapers idempotentes, com cache, retry, e
   tratamento de mudanças no HTML. Workers rodam automaticamente.

3. **Dataset massivo antes do modelo**: melhor ter 100K publicações
   bem coletadas que treinar em 1K. Quantidade de dados > sofisticação
   do modelo, especialmente no início.

4. **Ground truth real**: nada de matches inventados. Toda relação
   no modelo de treinamento vem de eventos observáveis (publicações
   conjuntas, citações, licenciamentos).

5. **Open source**: código aberto desde o dia um. Dataset publicado
   no HuggingFace quando estiver consolidado.

## Relação com o Argos

Ágora **by** Argos: é um spinoff que herda branding, paleta visual
(roxo #6B21A8, dark #1A1329, gold #D4A017) e referências ao olho de
Panoptes. Mas é um produto independente com foco diferente:

- **Argos**: gerencia o que JÁ É PI
- **Ágora**: descobre o que PODE virar valor (PI, parceria, projeto)

Os dois podem coexistir e trocar dados via API.

## Licença

Software livre (MIT). Dataset Ágora será publicado em CC-BY-4.0.
