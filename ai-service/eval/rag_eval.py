#!/usr/bin/env python3
"""
rag_eval.py — avaliação manual do Oráculo (RAG).

Executa perguntas pré-definidas contra /api/chat e imprime:
  - Pergunta
  - Resposta gerada
  - Fontes retornadas
  - Ground truth esperado (para avaliação manual: 👍 ou 👎)

Uso:
  python3 eval/rag_eval.py
  python3 eval/rag_eval.py --api http://localhost:8081
"""

import argparse
import json
import os
import sys

import requests

API_URL = os.getenv("NEXT_PUBLIC_API_URL", "http://localhost:8081")

EVAL_CASES = [
    {
        "question": "Quem trabalha com nanopartículas na UFV?",
        "expected_keywords": ["nano", "DFP", "biológico"],
    },
    {
        "question": "Quais patentes do DFP foram concedidas recentemente?",
        "expected_keywords": ["DFP", "patente", "conced"],
    },
    {
        "question": "Há editais abertos para biotecnologia?",
        "expected_keywords": ["edital", "biotecnolog", "FAPEMIG", "FINEP", "CNPq"],
    },
    {
        "question": "Quais são as 5 publicações mais citadas da UFV?",
        "expected_keywords": ["publicaç", "citad"],
    },
    {
        "question": "Pesquisadores que trabalham com biocontrole de pragas",
        "expected_keywords": ["biocontrol", "praga", "UFV"],
    },
    {
        "question": "Quais grupos de pesquisa atuam em energia renovável?",
        "expected_keywords": ["energia", "renovável", "grupo"],
    },
    {
        "question": "Há oportunidades de financiamento para pesquisa em saúde?",
        "expected_keywords": ["saúde", "financiam", "edital"],
    },
    {
        "question": "Quais áreas têm maior número de publicações nos últimos 5 anos?",
        "expected_keywords": ["publicaç", "área"],
    },
    {
        "question": "Brasil importa muita tecnologia agrícola?",
        "expected_keywords": ["import", "agrícol", "tecnolog"],
    },
    {
        "question": "Tendências de mercado em inteligência artificial no Brasil",
        "expected_keywords": ["IA", "inteligência artificial", "tendência"],
    },
]


def run_query(api_url: str, question: str, conversation_id: str = None) -> dict:
    payload = {"message": question}
    if conversation_id:
        payload["conversation_id"] = conversation_id
    try:
        resp = requests.post(f"{api_url}/api/chat", json=payload, timeout=60)
        resp.raise_for_status()
        return resp.json()
    except Exception as e:
        return {"error": str(e)}


def check_keywords(text: str, keywords: list) -> list:
    text_lower = text.lower()
    return [kw for kw in keywords if kw.lower() in text_lower]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--api", default=API_URL)
    parser.add_argument("--verbose", action="store_true")
    args = parser.parse_args()

    print(f"\n{'='*60}")
    print(f"AVALIAÇÃO DO ORÁCULO — {len(EVAL_CASES)} perguntas")
    print(f"API: {args.api}")
    print(f"{'='*60}\n")

    scores = []
    conversation_id = None

    for i, case in enumerate(EVAL_CASES, 1):
        print(f"\n[{i}/{len(EVAL_CASES)}] {case['question']}")
        print("-" * 50)

        result = run_query(args.api, case["question"], conversation_id)

        if "error" in result:
            print(f"  ❌ ERRO: {result['error']}")
            scores.append(False)
            continue

        # Reutilizar mesma conversa (opcional)
        conversation_id = result.get("conversation_id")

        answer   = result.get("message", "")
        sources  = result.get("sources", [])
        cost_usd = result.get("cost_usd", 0)

        print(f"\n  RESPOSTA:\n  {answer[:500]}{'...' if len(answer) > 500 else ''}")

        if sources:
            print(f"\n  FONTES ({len(sources)}):")
            for s in sources[:3]:
                print(f"    [{s['index']}] [{s['source_type']}] {s['title'][:70]}")

        # Verificação automática de keywords
        found = check_keywords(answer, case["expected_keywords"])
        hit = len(found) > 0
        scores.append(hit)

        status = "✅" if hit else "⚠️ "
        print(f"\n  {status} Keywords encontradas: {found or 'nenhuma'}")
        print(f"  💰 Custo: ${cost_usd:.6f}")

        if args.verbose:
            print(f"\n  DEBUG: {json.dumps(result, indent=2, ensure_ascii=False)[:1000]}")

    # Resumo
    total  = len(scores)
    passed = sum(scores)
    pct    = 100 * passed / total if total else 0

    print(f"\n{'='*60}")
    print(f"RESULTADO: {passed}/{total} ({pct:.0f}%)")
    print(f"{'='*60}\n")

    if pct < 50:
        print("⚠️  RAG com qualidade baixa. Verifique:")
        print("   1. Se os dados estão ingeridos (make ingest-all)")
        print("   2. Se os embeddings foram gerados (make generate-embeddings)")
        print("   3. Se o embed_server está rodando (make embed-server)")
        sys.exit(1)


if __name__ == "__main__":
    main()
