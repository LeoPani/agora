#!/usr/bin/env python3
"""
FINEP Editais Collector — Coleta chamadas públicas da FINEP.
Output: ai-service/data/editais_finep.jsonl
"""

import requests, json, re, time
from pathlib import Path
from bs4 import BeautifulSoup
from datetime import datetime

OUTPUT_DIR  = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
OUTPUT_FILE = OUTPUT_DIR / "editais_finep.jsonl"

DELAY   = 2.0
HEADERS = {"User-Agent": "Agora/1.0 (UFV NIT)"}

# FINEP tem API REST pública para chamadas
API_URL  = "https://www.finep.gov.br/api/chamadas_publicas"
LIST_URL = "https://www.finep.gov.br/chamadas-publicas"

def fetch_api() -> list[dict]:
    """Tenta API REST da FINEP."""
    try:
        r = requests.get(API_URL, headers=HEADERS, timeout=20)
        if r.status_code == 200:
            data = r.json()
            items = data if isinstance(data, list) else data.get("data", [])
            if items:
                print(f"  FINEP API: {len(items)} chamadas")
                return items
    except Exception:
        pass
    return []

def fetch_web() -> list[dict]:
    """Scraping da página de chamadas da FINEP."""
    try:
        r = requests.get(LIST_URL, headers=HEADERS, timeout=20)
        r.raise_for_status()
    except Exception as e:
        print(f"  FINEP web: {e}")
        return []

    soup  = BeautifulSoup(r.text, "html.parser")
    items = []

    rows = (soup.select("table.chamadas-publicas tbody tr")
            or soup.select(".chamada-item, .views-row, article"))

    for row in rows:
        cells = row.find_all("td") if row.name == "tr" else []
        if cells:
            title    = cells[0].get_text(strip=True) if cells else ""
            deadline = cells[1].get_text(strip=True) if len(cells) > 1 else ""
            status   = cells[2].get_text(strip=True) if len(cells) > 2 else ""
            link_el  = row.select_one("a[href]")
            link     = link_el["href"] if link_el else ""
        else:
            title_el  = row.select_one("h2, h3, .title")
            title     = title_el.get_text(strip=True) if title_el else ""
            date_el   = row.select_one(".date, time, .prazo")
            deadline  = date_el.get_text(strip=True) if date_el else ""
            status    = "aberto"
            link_el   = row.select_one("a[href]")
            link      = link_el["href"] if link_el else ""

        if not title:
            continue
        if link and not link.startswith("http"):
            link = "https://www.finep.gov.br" + link

        items.append({
            "source":      "FINEP",
            "external_id": link or title[:80],
            "title":       title,
            "description": "",
            "url":         link,
            "deadline":    deadline,
            "status":      status.lower() if status else "aberto",
            "raw_data":    {"scraped_at": datetime.now().isoformat()},
        })

    print(f"  FINEP web: {len(items)} chamadas")
    return items

def normalize_api_item(raw: dict) -> dict:
    return {
        "source":      "FINEP",
        "external_id": str(raw.get("id") or raw.get("codigo") or raw.get("titulo", "")[:80]),
        "title":       raw.get("titulo") or raw.get("nome") or "",
        "description": raw.get("descricao") or raw.get("objeto") or "",
        "url":         raw.get("url") or raw.get("link") or "",
        "deadline":    str(raw.get("data_encerramento") or raw.get("prazo") or ""),
        "status":      (raw.get("situacao") or raw.get("status") or "aberto").lower(),
        "raw_data":    raw,
    }

def collect():
    print("Coletando editais FINEP...")
    raw = fetch_api()
    if raw:
        items = [normalize_api_item(r) for r in raw if r.get("titulo") or r.get("nome")]
    else:
        time.sleep(DELAY)
        items = fetch_web()

    with open(OUTPUT_FILE, "w") as f:
        for it in items:
            f.write(json.dumps(it, ensure_ascii=False, default=str) + "\n")

    print(f"FINEP: {len(items)} editais salvos")

if __name__ == "__main__":
    collect()
