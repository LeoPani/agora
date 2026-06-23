#!/usr/bin/env python3
"""
CNPq Chamadas Collector — Coleta chamadas públicas do CNPq.
Output: ai-service/data/editais_cnpq.jsonl
"""

import requests, json, re, time
from pathlib import Path
from bs4 import BeautifulSoup
from datetime import datetime

OUTPUT_DIR  = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
OUTPUT_FILE = OUTPUT_DIR / "editais_cnpq.jsonl"

DELAY   = 2.0
HEADERS = {"User-Agent": "Agora/1.0 (UFV NIT)"}

# CNPq tem feed de chamadas
URLS = [
    "https://www.gov.br/cnpq/pt-br/acesso-a-informacao/acoes-e-programas/chamadas-publicas",
    "https://www.cnpq.br/web/guest/chamadas-publicas",
]

def fetch(url: str) -> str | None:
    try:
        r = requests.get(url, headers=HEADERS, timeout=25)
        r.raise_for_status()
        return r.text
    except Exception as e:
        print(f"  CNPq fetch {url}: {e}")
        return None

def parse_gov_br(html: str) -> list[dict]:
    """Página gov.br do CNPq."""
    soup  = BeautifulSoup(html, "html.parser")
    items = []

    # Estrutura: lista de artigos / tiles
    articles = (soup.select("article.tileItem, .listing-item, li.chamada")
                or soup.select("div.chamada-item, .views-row"))

    for art in articles:
        title_el   = art.select_one("h2, h3, .tileHeadline, .title")
        link_el    = art.select_one("a[href]")
        date_el    = art.select_one(".date, time, .tileBody .date-display-single")
        desc_el    = art.select_one("p, .description, .tileBody")
        status_el  = art.select_one(".label, .status, .badge")

        title = title_el.get_text(strip=True) if title_el else ""
        if not title:
            continue

        link = link_el["href"] if link_el else ""
        if link and link.startswith("/"):
            link = "https://www.gov.br" + link

        items.append({
            "source":      "CNPq",
            "external_id": link or title[:80],
            "title":       title,
            "description": desc_el.get_text(strip=True)[:500] if desc_el else "",
            "url":         link,
            "deadline":    date_el.get_text(strip=True) if date_el else "",
            "status":      status_el.get_text(strip=True).lower() if status_el else "aberto",
            "raw_data":    {"scraped_at": datetime.now().isoformat()},
        })

    return items

def collect():
    print("Coletando chamadas CNPq...")
    all_items = []

    for url in URLS:
        html = fetch(url)
        if html:
            items = parse_gov_br(html)
            print(f"  {url}: {len(items)} chamadas")
            all_items.extend(items)
        time.sleep(DELAY)

    seen = set()
    unique = []
    for it in all_items:
        k = it["external_id"]
        if k not in seen:
            seen.add(k)
            unique.append(it)

    with open(OUTPUT_FILE, "w") as f:
        for it in unique:
            f.write(json.dumps(it, ensure_ascii=False, default=str) + "\n")

    print(f"CNPq: {len(unique)} chamadas salvas")

if __name__ == "__main__":
    collect()
