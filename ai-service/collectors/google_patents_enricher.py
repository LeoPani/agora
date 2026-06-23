#!/usr/bin/env python3
"""
Google Patents Enricher — Enriquece patentes UFV com texto completo.

Busca reivindicações, descrição, citações e família internacional
para cada patente UFV identificada no banco via Google Patents.

ATENÇÃO: Google Patents tem captcha ocasional.
- Rate limit: 1 request a cada 3s
- Após 10 falhas seguidas: pausa de 60s
- Salva checkpoint por numero_pedido

Output:
- ai-service/data/google_patents_enriched.jsonl

Dependências: pip install requests beautifulsoup4
"""

import requests
import json
import time
import re
import sys
import unicodedata
from pathlib import Path
from urllib.parse import quote

OUTPUT_DIR = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

CHECKPOINT_FILE = OUTPUT_DIR / "google_patents_checkpoint.json"
OUTPUT_FILE     = OUTPUT_DIR / "google_patents_enriched.jsonl"

DELAY          = 3.0
MAX_FAILURES   = 10
PAUSE_ON_FAIL  = 60

HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/120.0.0.0 Safari/537.36"
    ),
    "Accept-Language": "pt-BR,pt;q=0.9,en;q=0.8",
}


def load_ufv_numbers_from_jsonl() -> list[str]:
    """Lê os arquivos INPI JSONL e extrai números de patentes UFV."""
    numbers = []
    data_dir = Path(__file__).parent.parent / "data"
    for path in sorted(data_dir.glob("inpi_patents_part_*.jsonl")):
        with open(path) as f:
            for line in f:
                if not line.strip():
                    continue
                try:
                    rec = json.loads(line)
                    if rec.get("is_ufv") and rec.get("inpi_number"):
                        numbers.append(rec["inpi_number"])
                except Exception:
                    pass
    return numbers


def load_checkpoint() -> set[str]:
    if CHECKPOINT_FILE.exists():
        with open(CHECKPOINT_FILE) as f:
            return set(json.load(f).get("done", []))
    return set()


def save_checkpoint(done: set[str]):
    with open(CHECKPOINT_FILE, "w") as f:
        json.dump({"done": list(done)}, f)


def fetch_patent(inpi_number: str) -> dict | None:
    """Busca dados de uma patente no Google Patents."""
    # Formato BR: BRXXXXXXXXXXXA (letras e números)
    # Google Patents aceita "BR" + número
    number_clean = inpi_number.strip().upper()
    if not number_clean.startswith("BR"):
        number_clean = "BR" + number_clean

    url = f"https://patents.google.com/patent/{number_clean}/pt"

    try:
        r = requests.get(url, headers=HEADERS, timeout=30)
        if r.status_code == 404:
            return {"inpi_number": inpi_number, "found": False}
        if r.status_code == 429 or "captcha" in r.text.lower():
            print(f"  [CAPTCHA/429] {inpi_number} — aguardando 120s...")
            time.sleep(120)
            return None
        r.raise_for_status()
    except requests.RequestException as e:
        print(f"  [ERRO] {inpi_number}: {e}")
        return None

    return parse_patent_page(inpi_number, r.text)


def parse_patent_page(inpi_number: str, html: str) -> dict:
    """Extrai dados da página HTML do Google Patents."""
    from bs4 import BeautifulSoup
    soup = BeautifulSoup(html, "html.parser")

    def text_of(selector: str) -> str:
        el = soup.select_one(selector)
        return el.get_text(separator=" ", strip=True) if el else ""

    def texts_of(selector: str) -> list[str]:
        return [el.get_text(strip=True) for el in soup.select(selector) if el.get_text(strip=True)]

    # Reivindicações
    claims_els = soup.select(".claims-text, [itemprop='claims'], .claim")
    claims = " ".join(el.get_text(separator=" ", strip=True) for el in claims_els)

    # Descrição
    desc_els = soup.select(".description-paragraph, [itemprop='description'] p")
    description = " ".join(el.get_text(separator=" ", strip=True) for el in desc_els[:50])

    # Citações de patentes
    cited_patents = texts_of(".citation a[href*='/patent/']")

    # Citações NPL (non-patent literature)
    cited_papers = texts_of(".npl-text, .citation-npl")

    # Família
    family_els = soup.select(".family-table a, .application-number a")
    family_members = list({el.get_text(strip=True) for el in family_els
                           if el.get_text(strip=True) and len(el.get_text(strip=True)) > 3})

    # Título e resumo (backup)
    title   = text_of("h1#title, [itemprop='name']")
    abstract = text_of("[itemprop='abstract'] .abstract, .abstract-text")

    return {
        "inpi_number":    inpi_number,
        "found":          True,
        "title":          title,
        "abstract":       abstract,
        "claims":         claims[:10000] if claims else None,
        "description":    description[:20000] if description else None,
        "family_members": family_members[:20],
        "cited_patents":  cited_patents[:50],
        "cited_papers":   cited_papers[:50],
    }


def run():
    try:
        from bs4 import BeautifulSoup  # noqa: F401
    except ImportError:
        print("ERRO: instale: pip install beautifulsoup4")
        sys.exit(1)

    numbers = load_ufv_numbers_from_jsonl()
    if not numbers:
        print("Nenhuma patente UFV encontrada nos arquivos JSONL.")
        print("Execute primeiro: make collect-inpi")
        sys.exit(1)

    print(f"Patentes UFV encontradas: {len(numbers)}")

    done = load_checkpoint()
    remaining = [n for n in numbers if n not in done]
    print(f"Já processadas: {len(done)} | Restantes: {len(remaining)}")

    failures = 0
    processed = 0

    with open(OUTPUT_FILE, "a") as out_f:
        for i, number in enumerate(remaining):
            print(f"[{i+1}/{len(remaining)}] {number}", end=" ", flush=True)

            result = fetch_patent(number)
            if result is None:
                failures += 1
                print(f"FALHOU ({failures}/{MAX_FAILURES})")
                if failures >= MAX_FAILURES:
                    print(f"  Muitas falhas seguidas. Aguardando {PAUSE_ON_FAIL}s...")
                    time.sleep(PAUSE_ON_FAIL)
                    failures = 0
                continue

            failures = 0
            found_str = "OK" if result.get("found") else "404"
            print(found_str)

            out_f.write(json.dumps(result, ensure_ascii=False) + "\n")
            out_f.flush()
            done.add(number)
            processed += 1

            if processed % 20 == 0:
                save_checkpoint(done)

            time.sleep(DELAY)

    save_checkpoint(done)
    print(f"\nEnriquecimento finalizado: {processed} patentes processadas")


if __name__ == "__main__":
    run()
