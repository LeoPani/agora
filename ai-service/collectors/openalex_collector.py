#!/usr/bin/env python3
"""
OpenAlex Collector — Coleta TODAS as publicações da UFV via OpenAlex API.

OpenAlex é gratuito, sem chave de API, 100K requests/dia.
Use mailto no parâmetro para entrar no polite pool.

UFV no OpenAlex: I4310312296

Output:
- ai-service/data/openalex_publications.jsonl
- ai-service/data/openalex_researchers.jsonl
- ai-service/data/openalex_coauthorships.jsonl
"""

import requests
import json
import time
import unicodedata
from pathlib import Path

UFV_OPENALEX_ID = "I146165071"
BASE_URL = "https://api.openalex.org"
MAILTO = "agora@argos.dev"
OUTPUT_DIR = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)


def normalize_name(name: str) -> str:
    nfkd = unicodedata.normalize('NFKD', name)
    ascii_str = ''.join(c for c in nfkd if not unicodedata.combining(c))
    return ' '.join(ascii_str.lower().split())


def fetch_page(cursor: str = "*", per_page: int = 200) -> dict:
    url = f"{BASE_URL}/works"
    params = {
        "filter": f"institutions.id:{UFV_OPENALEX_ID}",
        "per-page": per_page,
        "cursor": cursor,
        "mailto": MAILTO,
    }
    r = requests.get(url, params=params, timeout=30)
    r.raise_for_status()
    return r.json()


def _reconstruct_abstract(inverted_index):
    if not inverted_index:
        return None
    positions = {}
    for word, idxs in inverted_index.items():
        for idx in idxs:
            positions[idx] = word
    return " ".join(positions[i] for i in sorted(positions.keys()))


def collect_all():
    publications = []
    researchers_seen = {}
    coauthorships = []

    cursor = "*"
    page = 0
    total = 0

    print(f"Iniciando coleta OpenAlex UFV ({UFV_OPENALEX_ID})...")

    while cursor:
        page += 1
        try:
            data = fetch_page(cursor=cursor)
        except Exception as e:
            print(f"Erro página {page}: {e}")
            time.sleep(5)
            continue

        results = data.get("results", [])
        if not results:
            break

        for work in results:
            pub = {
                "openalex_id": work["id"].replace("https://openalex.org/", ""),
                "doi": work.get("doi"),
                "title": work.get("title") or "",
                "abstract": _reconstruct_abstract(work.get("abstract_inverted_index")),
                "publication_year": work.get("publication_year"),
                "publication_date": work.get("publication_date"),
                "type": work.get("type"),
                "cited_by_count": work.get("cited_by_count", 0),
                "topics": [
                    {
                        "id": (t.get("id") or "").replace("https://openalex.org/", ""),
                        "name": t.get("display_name"),
                        "score": t.get("score"),
                    }
                    for t in (work.get("topics") or [])
                ],
            }
            publications.append(pub)

            for idx, authorship in enumerate(work.get("authorships", []) or []):
                author = authorship.get("author") or {}
                author_id = (author.get("id") or "").replace("https://openalex.org/", "")
                full_name = author.get("display_name") or ""
                norm = normalize_name(full_name)
                if not norm:
                    continue

                institutions = authorship.get("institutions") or []
                is_ufv = any(
                    (i.get("id") or "").endswith(UFV_OPENALEX_ID)
                    for i in institutions
                )
                if not is_ufv:
                    continue

                if norm not in researchers_seen:
                    researchers_seen[norm] = {
                        "openalex_id": author_id,
                        "orcid": author.get("orcid"),
                        "full_name": full_name,
                        "normalized_name": norm,
                        "department": None,
                    }

                coauthorships.append({
                    "publication_openalex_id": pub["openalex_id"],
                    "researcher_normalized_name": norm,
                    "position": idx + 1,
                })

        total += len(results)
        cursor = data.get("meta", {}).get("next_cursor")

        if page % 5 == 0:
            print(f"  página {page}: {total} pubs | {len(researchers_seen)} pesquisadores")

        time.sleep(0.1)

    with open(OUTPUT_DIR / "openalex_publications.jsonl", "w") as f:
        for pub in publications:
            f.write(json.dumps(pub, ensure_ascii=False) + "\n")

    with open(OUTPUT_DIR / "openalex_researchers.jsonl", "w") as f:
        for r in researchers_seen.values():
            f.write(json.dumps(r, ensure_ascii=False) + "\n")

    with open(OUTPUT_DIR / "openalex_coauthorships.jsonl", "w") as f:
        for ca in coauthorships:
            f.write(json.dumps(ca, ensure_ascii=False) + "\n")

    print(f"\nColeta finalizada:")
    print(f"  {len(publications)} publicações")
    print(f"  {len(researchers_seen)} pesquisadores únicos da UFV")
    print(f"  {len(coauthorships)} co-autorias")


if __name__ == "__main__":
    collect_all()
