#!/usr/bin/env python3
"""
EMBRAPII Chamadas Collector — Coleta chamadas e projetos da EMBRAPII.
Output: ai-service/data/editais_embrapii.jsonl
"""

import requests, json, time
from pathlib import Path
from bs4 import BeautifulSoup
from datetime import datetime

OUTPUT_DIR  = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
OUTPUT_FILE = OUTPUT_DIR / "editais_embrapii.jsonl"

DELAY   = 2.0
HEADERS = {"User-Agent": "Agora/1.0 (UFV NIT)"}

# EMBRAPII tem API pública e página de chamadas
API_URLS = [
    "https://embrapii.org.br/wp-json/wp/v2/posts?categories=chamadas&per_page=100",
    "https://embrapii.org.br/wp-json/wp/v2/posts?tags=edital&per_page=100",
]
LIST_URL = "https://embrapii.org.br/categorias/chamadas/"

def fetch_wp_api() -> list[dict]:
    """EMBRAPII usa WordPress com REST API."""
    items = []
    for url in API_URLS:
        try:
            r = requests.get(url, headers=HEADERS, timeout=20)
            if r.status_code == 200:
                posts = r.json()
                if isinstance(posts, list):
                    items.extend(posts)
                    print(f"  EMBRAPII WP API: {len(posts)} posts via {url}")
        except Exception as e:
            print(f"  EMBRAPII API {url}: {e}")
        time.sleep(1)
    return items

def normalize_wp_post(post: dict) -> dict:
    from bs4 import BeautifulSoup as BS
    title = (post.get("title", {}) or {}).get("rendered", "") or ""
    title = BS(title, "html.parser").get_text(strip=True)

    content = (post.get("excerpt", {}) or {}).get("rendered", "") or ""
    desc    = BS(content, "html.parser").get_text(strip=True)[:500]

    link  = post.get("link", "")
    date  = post.get("date", "")[:10] if post.get("date") else ""

    return {
        "source":      "EMBRAPII",
        "external_id": str(post.get("id") or link or title[:80]),
        "title":       title,
        "description": desc,
        "url":         link,
        "deadline":    "",
        "status":      "aberto",
        "raw_data":    {"wp_id": post.get("id"), "date": date},
    }

def fetch_web() -> list[dict]:
    """Fallback: scraping da listagem."""
    try:
        r = requests.get(LIST_URL, headers=HEADERS, timeout=20)
        r.raise_for_status()
    except Exception as e:
        print(f"  EMBRAPII web: {e}")
        return []

    soup  = BeautifulSoup(r.text, "html.parser")
    items = []

    for art in soup.select("article, .post-item, .entry"):
        title_el = art.select_one("h2, h3, .entry-title")
        link_el  = art.select_one("a[href]")
        date_el  = art.select_one("time, .date")
        desc_el  = art.select_one("p, .excerpt")

        title = title_el.get_text(strip=True) if title_el else ""
        if not title:
            continue

        link = link_el["href"] if link_el else ""
        items.append({
            "source":      "EMBRAPII",
            "external_id": link or title[:80],
            "title":       title,
            "description": desc_el.get_text(strip=True)[:500] if desc_el else "",
            "url":         link,
            "deadline":    date_el.get_text(strip=True) if date_el else "",
            "status":      "aberto",
            "raw_data":    {"scraped_at": datetime.now().isoformat()},
        })

    print(f"  EMBRAPII web: {len(items)} posts")
    return items

def collect():
    print("Coletando chamadas EMBRAPII...")

    raw = fetch_wp_api()
    if raw:
        items = [normalize_wp_post(p) for p in raw
                 if (p.get("title", {}) or {}).get("rendered")]
    else:
        items = fetch_web()

    # Deduplicar
    seen, unique = set(), []
    for it in items:
        k = it["external_id"]
        if k not in seen:
            seen.add(k)
            unique.append(it)

    with open(OUTPUT_FILE, "w") as f:
        for it in unique:
            f.write(json.dumps(it, ensure_ascii=False, default=str) + "\n")

    print(f"EMBRAPII: {len(unique)} chamadas salvas")

if __name__ == "__main__":
    collect()
