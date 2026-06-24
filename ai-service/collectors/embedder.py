#!/usr/bin/env python3
"""
Embedder — Gera vetores semânticos para publicações e patentes da UFV.

Modelo: paraphrase-multilingual-MiniLM-L12-v2 (384 dims, ~120MB)
  - Bom suporte a português
  - Rápido (~500 docs/s em CPU)
  - Open source, sem API key

Alternativa para patentes: --model LeoPani/patentbert-br (PatentBERT-BR)

Saída:
  data/embeddings_publications.jsonl  — {id, embedding: [384 floats]}
  data/embeddings_patents.jsonl       — {id, embedding: [384 floats]}

Uso:
  python3 embedder.py                          # publica + patentes
  python3 embedder.py --entity publications    # só publicações
  python3 embedder.py --entity patents         # só patentes
  python3 embedder.py --model LeoPani/patentbert-br --entity patents
  python3 embedder.py --batch 128 --limit 5000
"""

import argparse
import json
import os
import sys
import time
from pathlib import Path

DATA_DIR = Path(__file__).parent.parent / "data"
DATA_DIR.mkdir(parents=True, exist_ok=True)

DEFAULT_MODEL = "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2"
DB_URL = os.getenv("DATABASE_URL", "postgresql://agora:agora_dev@localhost:5433/agora")


def get_db():
    import psycopg2
    return psycopg2.connect(DB_URL)


def load_model(model_name: str):
    try:
        from sentence_transformers import SentenceTransformer
    except ImportError:
        print("Instalando sentence-transformers...")
        os.system(f"{sys.executable} -m pip install sentence-transformers -q")
        from sentence_transformers import SentenceTransformer

    print(f"Carregando modelo: {model_name}")
    t0 = time.time()
    model = SentenceTransformer(model_name)
    print(f"  modelo pronto em {time.time()-t0:.1f}s")
    return model


def make_text_publication(row) -> str:
    """Concatena campos relevantes para embedding de publicação."""
    parts = []
    if row.get("title"):
        parts.append(row["title"])
    if row.get("abstract"):
        parts.append(row["abstract"][:1000])
    topics = row.get("topics") or []
    if topics:
        names = [t.get("name") or t if isinstance(t, str) else "" for t in topics[:5]]
        parts.append(" | ".join(filter(None, names)))
    return ". ".join(filter(None, parts)) or row.get("title") or ""


def make_text_patent(row) -> str:
    """Concatena campos relevantes para embedding de patente."""
    parts = []
    if row.get("title"):
        parts.append(row["title"])
    if row.get("abstract"):
        parts.append(row["abstract"][:1000])
    if row.get("claims"):
        parts.append(str(row["claims"])[:500])
    ipc = row.get("ipc_codes") or []
    if ipc:
        parts.append(" ".join(ipc[:4]))
    return ". ".join(filter(None, parts)) or row.get("title") or ""


def embed_entity(entity: str, model, batch_size: int, limit: int | None):
    conn = get_db()
    cur  = conn.cursor()

    if entity == "publications":
        cur.execute("""
            SELECT id, title, abstract, topics
            FROM publications
            WHERE embedding IS NULL
            ORDER BY id
            LIMIT %s
        """, (limit or 1_000_000,))
        make_text = make_text_publication
        out_file  = DATA_DIR / "embeddings_publications.jsonl"
    else:
        cur.execute("""
            SELECT id, title, abstract, ipc_codes, claims
            FROM patents
            WHERE embedding IS NULL
            ORDER BY id
            LIMIT %s
        """, (limit or 1_000_000,))
        make_text = make_text_patent
        out_file  = DATA_DIR / "embeddings_patents.jsonl"

    rows = cur.fetchall()
    cur.close()
    conn.close()

    if not rows:
        print(f"  {entity}: nenhum registro sem embedding.")
        return 0

    cols = [d[0] for d in cur.description] if hasattr(cur, 'description') else []

    # Reconstrói como dicts
    def row_to_dict(row):
        keys = ["id", "title", "abstract",
                "topics" if entity == "publications" else "ipc_codes",
                "claims" if entity == "patents" else None]
        keys = [k for k in keys if k]
        return dict(zip(keys, row))

    records = [row_to_dict(r) for r in rows]
    texts   = [make_text(r) for r in records]
    ids     = [r["id"] for r in records]

    print(f"  {entity}: {len(records)} registros para embedar")

    total_written = 0
    with open(out_file, "w") as f:
        for i in range(0, len(texts), batch_size):
            batch_texts = texts[i:i+batch_size]
            batch_ids   = ids[i:i+batch_size]

            vecs = model.encode(batch_texts, show_progress_bar=False,
                                normalize_embeddings=True)

            for rec_id, vec in zip(batch_ids, vecs):
                f.write(json.dumps({"id": rec_id, "embedding": vec.tolist()}) + "\n")

            total_written += len(batch_ids)
            pct = total_written / len(records) * 100
            print(f"    {total_written}/{len(records)} ({pct:.0f}%) — batch {i//batch_size+1}", end="\r")

    print(f"\n  {entity}: {total_written} embeddings → {out_file.name}")
    return total_written


def main():
    parser = argparse.ArgumentParser(description="Embedder semântico Ágora")
    parser.add_argument("--entity", choices=["publications", "patents", "all"], default="all")
    parser.add_argument("--model",  default=DEFAULT_MODEL)
    parser.add_argument("--batch",  type=int, default=64)
    parser.add_argument("--limit",  type=int, default=None, help="Limite de registros por entidade")
    args = parser.parse_args()

    model = load_model(args.model)

    entities = ["publications", "patents"] if args.entity == "all" else [args.entity]
    total = 0
    for ent in entities:
        n = embed_entity(ent, model, args.batch, args.limit)
        total += n

    print(f"\nTotal: {total} embeddings gerados.")


if __name__ == "__main__":
    main()
