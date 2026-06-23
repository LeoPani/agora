#!/usr/bin/env python3
"""
FAPEMIG Editais Collector — Coleta chamadas abertas e recentes da FAPEMIG.
Output: ai-service/data/editais_fapemig.jsonl
"""

import requests, json, re, time
from pathlib import Path
from bs4 import BeautifulSoup
from datetime import datetime

OUTPUT_DIR  = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
OUTPUT_FILE = OUTPUT_DIR / "editais_fapemig.jsonl"

DELAY   = 2.0
HEADERS = {"User-Agent": "Agora/1.0 (UFV NIT)"}

URLS = [
    "https://fapemig.br/pt/chamadas/",
    "https://fapemig.br/pt/editais/",
]

def fetch(url: str) -> str | None:
    try:
        r = requests.get(url, headers=HEADERS, timeout=20)
        r.raise_for_status()
        return r.text
    except Exception as e:
        print(f"  FAPEMIG fetch {url}: {e}")
        return None

def parse_page(html: str, base_url: str) -> list[dict]:
    soup  = BeautifulSoup(html, "html.parser")
    items = []

    # Estrutura FAPEMIG: cards de chamadas
    cards = (soup.select(".card-chamada, .chamada-item, article.post")
             or soup.select("div.item-edital, li.edital"))

    for card in cards:
        title_el   = card.select_one("h2, h3, h4, .title, .entry-title")
        link_el    = card.select_one("a[href]")
        date_el    = card.select_one(".date, time, .data-publicacao, .prazo")
        status_el  = card.select_one(".status, .situacao, .badge")
        desc_el    = card.select_one("p, .excerpt, .resumo")

        title = title_el.get_text(strip=True) if title_el else ""
        if not title:
            continue

        link = link_el["href"] if link_el else ""
        if link and not link.startswith("http"):
            link = base_url.rstrip("/") + "/" + link.lstrip("/")

        deadline = date_el.get_text(strip=True) if date_el else ""
        status   = status_el.get_text(strip=True) if status_el else "aberto"
        desc     = desc_el.get_text(strip=True)[:500] if desc_el else ""

        items.append({
            "source":       "FAPEMIG",
            "external_id":  link or title[:80],
            "title":        title,
            "description":  desc,
            "url":          link,
            "deadline":     deadline,
            "status":       status.lower() if status else "aberto",
            "raw_data":     {"scraped_at": datetime.now().isoformat()},
        })

    return items

def collect():
    print("Coletando editais FAPEMIG...")
    all_items = []
    for url in URLS:
        html = fetch(url)
        if html:
            items = parse_page(html, url)
            print(f"  {url}: {len(items)} editais")
            all_items.extend(items)
        time.sleep(DELAY)

    # Deduplicar por external_id
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

    print(f"FAPEMIG: {len(unique)} editais salvos")

if __name__ == "__main__":
    collect()
