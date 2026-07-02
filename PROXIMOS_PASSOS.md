# Próximos Passos — Ágora by Argos

Este documento consolida o plano de ação sugerido para corrigir os problemas atuais e implementar as melhorias e funcionalidades que estão faltando no ecossistema do **Ágora**.

---

## 🛠️ Fase 1: Infraestrutura e Resiliência (Imediato)

### 1. Inicializar o Ambiente Local
* **Objetivo**: Colocar o banco de dados PostgreSQL (com pgvector) e o servidor de embeddings local em execução.
* **Ações**:
  1. Inicie o daemon do Docker (ex: abrir o Docker Desktop no macOS).
  2. Suba o banco e as migrações:
     ```bash
     make db-up
     make migrate
     ```
  3. Valide o status das migrações:
     ```bash
     make migrate-status
     ```

### 2. Implementar Suíte de Testes
* **Objetivo**: Garantir que alterações no RAG e no buscador não introduzam regressões.
* **Ações**:
  - Criar o arquivo `retriever_test.go` sob [backend/internal/rag/](file:///Users/leopani/Projetos/agora/backend/internal/rag).
  - Validar a lógica de mesclagem RRF (Reciprocal Rank Fusion) e o comportamento de fallback lexical se o servidor de embeddings estiver offline.

### 3. Resiliência do Cliente de LLM
* **Objetivo**: Evitar respostas quebradas no Oráculo de chat devido a limites de requisição ou instabilidades na API do Groq.
* **Ações**:
  - Modificar [groq.go](file:///Users/leopani/Projetos/agora/backend/internal/llm/groq.go) para tratar status HTTP não-200.
  - Implementar uma política de retentativas automáticas (*retry* com *exponential backoff*).

---

## 🚀 Fase 2: Otimização do RAG e Busca Semântica (Médio Prazo)

### 1. Embeddings para Grupos de Pesquisa
* **Objetivo**: Permitir buscas semânticas (conceitos e sinônimos) sobre as competências e linhas de pesquisa dos grupos de pesquisa, não apenas busca textual rígida.
* **Ações**:
  1. Criar uma nova migração para adicionar a coluna `embedding vector(384)` na tabela `research_groups`.
  2. Atualizar o script [generate_embeddings.py](file:///Users/leopani/Projetos/agora/ai-service/embeddings/generate_embeddings.py) para carregar grupos de pesquisa, chamar a API do MiniLM para gerar os embeddings das linhas de pesquisa e salvá-los no banco.
  3. Atualizar a função `vectorSearch` em [retriever.go](file:///Users/leopani/Projetos/agora/backend/internal/rag/retriever.go) para incluir grupos de pesquisa nas buscas semânticas.

---

## 💻 Fase 3: Controle Operacional via Frontend (Longo Prazo)

### 1. Rota de Execução Manual de Geração de Sinais
* **Objetivo**: Permitir que o gestor do NIT recalcule os sinais sob demanda sem depender exclusivamente do agendador em background.
* **Ações**:
  - Criar um endpoint `POST /api/v1/signals/generate` no backend em [main.go](file:///Users/leopani/Projetos/agora/backend/cmd/api/main.go) que execute o subprocesso Python chamando [signal_engine.py](file:///Users/leopani/Projetos/agora/ai-service/signal_engine.py).

### 2. Controle dos Coletores no Painel do Frontend
* **Objetivo**: Integrar no painel administrativo do Next.js botões para disparar ou monitorar as execuções dos coletores ([collectors](file:///Users/leopani/Projetos/agora/ai-service/collectors)).
