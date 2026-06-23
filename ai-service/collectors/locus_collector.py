#!/usr/bin/env python3
"""
LOCUS Collector — Coleta TODAS as publicações da UFV via DSpace 8 REST API.

LOCUS é o repositório institucional da UFV (DSpace 8).
API base: https://locus.ufv.br/server/api

Output:
- ai-service/data/locus_publications.jsonl
- ai-service/data/locus_communities.jsonl  (cache da árvore communities→collections)

Salva checkpoint a cada 500 itens para retomar em caso de falha.
"""

import requests
import json
import time
import unicodedata
from pathlib import Path

BASE_URL   = "https://locus.ufv.br/server/api"
LOCUS_BASE = "https://locus.ufv.br"
OUTPUT_DIR = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

CHECKPOINT_FILE = OUTPUT_DIR / "locus_checkpoint.json"
PAGE_SIZE       = 100
DELAY           = 1.0   # s entre requests
CHECKPOINT_EVERY = 500  # itens

HEADERS = {
    "User-Agent": "Agora/1.0 (UFV NIT; agora@argos.dev)",
    "Accept":     "application/json",
}


def get(url: str, params: dict = None, retries: int = 3) -> dict | None:
    for attempt in range(retries):
        try:
            r = requests.get(url, params=params, headers=HEADERS, timeout=20)
            if r.status_code == 404:
                return None
            r.raise_for_status()
            return r.json()
        except Exception as e:
            if attempt == retries - 1:
                print(f"  [ERRO] {url}: {e}")
                return None
            time.sleep(2 ** attempt)
    return None


def paginate(url: str, extra_params: dict = None) -> list[dict]:
    """Pagina todos os resultados de um endpoint DSpace."""
    results = []
    page = 0
    while True:
        params = {"size": PAGE_SIZE, "page": page, **(extra_params or {})}
        data = get(url, params)
        if not data:
            break

        # DSpace 8 usa _embedded para os resultados
        embedded = data.get("_embedded", {})
        items = []
        for v in embedded.values():
            if isinstance(v, list):
                items = v
                break

        if not items:
            break

        results.extend(items)

        page_info = data.get("page", {})
        total_pages = page_info.get("totalPages", 1)
        page += 1
        if page >= total_pages:
            break
        time.sleep(DELAY)

    return results


def list_communities() -> list[dict]:
    print("Listando communities...")
    communities = paginate(f"{BASE_URL}/core/communities")
    print(f"  {len(communities)} communities encontradas")
    return communities


def list_collections_for_community(community_uuid: str) -> list[dict]:
    url = f"{BASE_URL}/core/communities/{community_uuid}/collections"
    return paginate(url) or []


def fetch_items_for_collection(collection_uuid: str) -> list[dict]:
    """Usa discover/search/objects para paginar itens de uma collection."""
    url = f"{BASE_URL}/discover/search/objects"
    items = []
    page = 0
    while True:
        params = {
            "dsoType":  "ITEM",
            "scope":    collection_uuid,
            "size":     PAGE_SIZE,
            "page":     page,
        }
        data = get(url, params)
        if not data:
            break

        objects = (
            data.get("_embedded", {})
                .get("searchResult", {})
                .get("_embedded", {})
                .get("objects", [])
        )
        if not objects:
            break

        for obj in objects:
            item = obj.get("_embedded", {}).get("indexableObject")
            if item:
                items.append(item)

        page_meta = (
            data.get("_embedded", {})
                .get("searchResult", {})
                .get("page", {})
        )
        total_pages = page_meta.get("totalPages", 1)
        page += 1
        if page >= total_pages:
            break
        time.sleep(DELAY)

    return items


def meta(item: dict, field: str) -> str:
    vals = item.get("metadata", {}).get(field, [])
    return vals[0].get("value", "") if vals else ""


def meta_list(item: dict, field: str) -> list[str]:
    return [v.get("value", "") for v in item.get("metadata", {}).get(field, [])]


def parse_item(item: dict, community_name: str, collection_name: str) -> dict | None:
    title = meta(item, "dc.title")
    if not title:
        return None

    handle = item.get("handle", "")
    uuid   = item.get("uuid", "")
    url    = f"{LOCUS_BASE}/handle/{handle}" if handle else f"{LOCUS_BASE}/items/{uuid}"

    issued = meta(item, "dc.date.issued") or meta(item, "dc.date.created") or ""
    year   = issued[:4] if len(issued) >= 4 else None

    pub_type_raw = meta(item, "dc.type").lower()
    type_map = {
        "doctoralthesis":   "tese",
        "masterthesis":     "dissertacao",
        "article":          "artigo",
        "conferenceobject": "artigo_congresso",
        "thesis":           "tese",
        "dissertation":     "dissertacao",
        "book":             "livro",
        "bookpart":         "capitulo_livro",
    }
    pub_type = type_map.get(pub_type_raw, pub_type_raw or "outro")

    return {
        "locus_uuid":       uuid,
        "handle":           handle,
        "title":            title.strip(),
        "abstract":         meta(item, "dc.description.abstract") or meta(item, "dc.description"),
        "authors":          meta_list(item, "dc.contributor.author") or meta_list(item, "dc.creator"),
        "advisor":          meta(item, "dc.contributor.advisor"),
        "publication_year": int(year) if year and year.isdigit() else None,
        "type":             pub_type,
        "department":       community_name,
        "collection":       collection_name,
        "subjects":         meta_list(item, "dc.subject"),
        "url":              url,
        "language":         meta(item, "dc.language.iso"),
    }


def load_checkpoint() -> tuple[set[str], list[dict]]:
    if CHECKPOINT_FILE.exists():
        with open(CHECKPOINT_FILE) as f:
            data = json.load(f)
        done_uuids = set(data.get("done_collection_uuids", []))
        partial    = data.get("partial_publications", [])
        print(f"Checkpoint: {len(done_uuids)} collections já processadas, {len(partial)} pubs salvas")
        return done_uuids, partial
    return set(), []


def save_checkpoint(done_uuids: set[str], publications: list[dict]):
    with open(CHECKPOINT_FILE, "w") as f:
        json.dump({
            "done_collection_uuids": list(done_uuids),
            "partial_publications":  publications,
        }, f, ensure_ascii=False)


def collect_all():
    done_uuids, publications = load_checkpoint()
    seen_handles = {p["handle"] for p in publications if p.get("handle")}

    communities = list_communities()

    # Salvar árvore de communities
    communities_out = []
    all_collections = []
    for comm in communities:
        comm_uuid = comm.get("uuid", "")
        comm_name = comm.get("name", "")
        colls = list_collections_for_community(comm_uuid)
        for col in colls:
            all_collections.append({
                "community_uuid": comm_uuid,
                "community_name": comm_name,
                "collection_uuid": col.get("uuid", ""),
                "collection_name": col.get("name", ""),
            })
        communities_out.append({
            "uuid": comm_uuid,
            "name": comm_name,
            "collections": [{"uuid": c.get("uuid",""), "name": c.get("name","")} for c in colls],
        })
        time.sleep(DELAY)

    with open(OUTPUT_DIR / "locus_communities.jsonl", "w") as f:
        for c in communities_out:
            f.write(json.dumps(c, ensure_ascii=False) + "\n")

    print(f"\n{len(all_collections)} collections encontradas. Iniciando coleta de itens...\n")

    total_new = 0
    for i, col in enumerate(all_collections):
        col_uuid = col["collection_uuid"]
        if col_uuid in done_uuids:
            continue

        print(f"[{i+1}/{len(all_collections)}] {col['community_name']} / {col['collection_name']}")
        items = fetch_items_for_collection(col_uuid)
        new_in_col = 0

        for item in items:
            parsed = parse_item(item, col["community_name"], col["collection_name"])
            if not parsed:
                continue
            handle = parsed.get("handle", "")
            if handle and handle in seen_handles:
                continue
            if handle:
                seen_handles.add(handle)
            publications.append(parsed)
            new_in_col += 1

        done_uuids.add(col_uuid)
        total_new += new_in_col
        print(f"  +{new_in_col} itens | total acumulado: {len(publications)}")

        if len(publications) % CHECKPOINT_EVERY < new_in_col or new_in_col > 0:
            save_checkpoint(done_uuids, publications)

        time.sleep(DELAY)

    # Salvar output final
    with open(OUTPUT_DIR / "locus_publications.jsonl", "w") as f:
        for pub in publications:
            f.write(json.dumps(pub, ensure_ascii=False) + "\n")

    # Remover checkpoint após conclusão bem-sucedida
    if CHECKPOINT_FILE.exists():
        CHECKPOINT_FILE.unlink()

    print(f"\nColeta LOCUS finalizada:")
    print(f"  {len(publications)} publicações")
    print(f"  {len(communities)} communities")
    print(f"  {len(all_collections)} collections")


if __name__ == "__main__":
    collect_all()
