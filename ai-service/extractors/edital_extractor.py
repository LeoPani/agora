#!/usr/bin/env python3
"""
Edital Extractor — extrai campos estruturados de editais (PDF ou HTML)
usando LLM via endpoint interno do backend Go.

Uso:
  python3 extractors/edital_extractor.py --pdf caminho.pdf
  python3 extractors/edital_extractor.py --url https://...
  python3 extractors/edital_extractor.py --all   # re-extrai todos os JSONL de editais

Input:  PDF ou HTML bruto
Output: dict validado com campos do edital
"""

import argparse
import json
import os
import sys
import time
from pathlib import Path
from typing import Optional

import requests

BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8081")
DATA_DIR    = Path(__file__).parent.parent / "data"

EDITAL_SCHEMA = {
    "titulo":                 "Título completo da chamada/edital",
    "orgao":                  "Órgão financiador (FAPEMIG, FINEP, CNPq, EMBRAPII, etc.)",
    "numero":                 "Número da chamada (ex: '10/2026') ou null",
    "modalidade":             "Tipo (Pesquisa, Inovação, Bolsa, Subvenção, etc.)",
    "valor_total_brl":        "Valor total disponível como número, sem R$ (ou null)",
    "valor_por_projeto_brl":  "Valor máximo por projeto como número (ou null)",
    "data_abertura":          "Data de abertura YYYY-MM-DD (ou null)",
    "data_encerramento":      "Data limite de submissão YYYY-MM-DD",
    "areas_foco":             "Lista de áreas/temas contemplados",
    "publico_alvo":           "Quem pode submeter (pesquisadores, empresas, etc.)",
    "requisitos_pesquisador": "Requisitos do proponente (ou null)",
    "duracao_meses":          "Duração máxima do projeto em meses (ou null)",
    "url_origem":             "URL da chamada (ou null)",
}

PROMPT_TEMPLATE = """Você é um especialista em editais de pesquisa e fomento brasileiros.
Analise o texto abaixo e extraia os campos em JSON.

REGRAS RÍGIDAS:
1. Retorne APENAS JSON válido, sem markdown, sem ```, sem texto antes/depois.
2. Use null para campos não encontrados — não invente.
3. Datas no formato YYYY-MM-DD.
4. Valores em BRL como número (sem R$, sem pontos separadores de milhar).
5. Listas como arrays JSON.

CAMPOS A EXTRAIR:
{schema}

TEXTO DO EDITAL:
\"\"\"{texto}\"\"\"

JSON:"""

REQUIRED_FIELDS = ["titulo", "orgao", "data_encerramento"]


def extract_from_pdf(pdf_path: str) -> Optional[dict]:
    try:
        import pdfplumber
    except ImportError:
        print("[edital_extractor] pdfplumber não instalado — pip install pdfplumber")
        return None

    with pdfplumber.open(pdf_path) as pdf:
        text = "\n".join(p.extract_text() or "" for p in pdf.pages)

    if len(text) < 200:
        print(f"[edital_extractor] PDF muito curto ({len(text)} chars): {pdf_path}")
        return None

    return _extract_from_text(text, url_origem=pdf_path)


def extract_from_html(url: str) -> Optional[dict]:
    try:
        resp = requests.get(url, timeout=30, headers={"User-Agent": "Mozilla/5.0"})
        resp.raise_for_status()
    except Exception as e:
        print(f"[edital_extractor] Erro ao buscar {url}: {e}")
        return None

    try:
        from bs4 import BeautifulSoup
        soup = BeautifulSoup(resp.text, "html.parser")
        for tag in soup(["script", "style", "nav", "footer", "header"]):
            tag.decompose()
        text = soup.get_text(separator="\n", strip=True)
    except ImportError:
        text = resp.text

    return _extract_from_text(text, url_origem=url)


def _extract_from_text(text: str, url_origem: str) -> Optional[dict]:
    # Trunca se muito longo (limite seguro de ~20K chars)
    if len(text) > 20_000:
        text = text[:15_000] + "\n\n[...truncado...]\n\n" + text[-5_000:]

    prompt = PROMPT_TEMPLATE.format(
        schema=json.dumps(EDITAL_SCHEMA, indent=2, ensure_ascii=False),
        texto=text,
    )

    raw = _call_llm(prompt)
    if not raw:
        return None

    try:
        data = json.loads(raw)
    except json.JSONDecodeError as e:
        print(f"[edital_extractor] JSON inválido: {e}\nResposta: {raw[:300]}")
        return None

    if not all(data.get(f) for f in REQUIRED_FIELDS):
        print(f"[edital_extractor] Campos obrigatórios ausentes em: {data}")
        return None

    if "url_origem" not in data or not data["url_origem"]:
        data["url_origem"] = url_origem

    return data


def _call_llm(prompt: str) -> Optional[str]:
    try:
        resp = requests.post(
            f"{BACKEND_URL}/internal/llm/complete",
            json={
                "purpose":      "extract_edital",
                "prompt":       prompt,
                "temperature":  0.1,
                "json_mode":    True,
                "provider_hint": "groq",
            },
            timeout=120,
        )
        resp.raise_for_status()
        return resp.json().get("text")
    except Exception as e:
        print(f"[edital_extractor] Erro ao chamar LLM: {e}")
        return None


def reextract_all():
    """Re-extrai todos os editais nos arquivos JSONL de data/editais_*.jsonl."""
    out_path = DATA_DIR / "editais_ai_extracted.jsonl"
    count = 0
    with open(out_path, "w") as fout:
        for jsonl_file in DATA_DIR.glob("editais_*.jsonl"):
            if "ai_extracted" in jsonl_file.name:
                continue
            print(f"[edital_extractor] Processando {jsonl_file.name}...")
            with open(jsonl_file) as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    record = json.loads(line)
                    url = record.get("url") or record.get("link")
                    if url:
                        result = extract_from_html(url)
                        if result:
                            fout.write(json.dumps(result, ensure_ascii=False) + "\n")
                            count += 1
                            time.sleep(1)  # rate limit gentil
    print(f"[edital_extractor] {count} editais extraídos → {out_path}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Edital Extractor com LLM")
    grp = parser.add_mutually_exclusive_group(required=True)
    grp.add_argument("--pdf", help="Caminho para PDF do edital")
    grp.add_argument("--url", help="URL da página do edital")
    grp.add_argument("--all", action="store_true", help="Re-extrair todos os editais")
    args = parser.parse_args()

    if args.all:
        reextract_all()
    elif args.pdf:
        result = extract_from_pdf(args.pdf)
        if result:
            print(json.dumps(result, indent=2, ensure_ascii=False))
        else:
            sys.exit(1)
    elif args.url:
        result = extract_from_html(args.url)
        if result:
            print(json.dumps(result, indent=2, ensure_ascii=False))
        else:
            sys.exit(1)
