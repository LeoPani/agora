#!/usr/bin/env python3
"""
generate_embeddings.py — gera embeddings semânticos para publicações,
patentes e oportunidades usando sentence-transformers.

Modelo: paraphrase-multilingual-MiniLM-L12-v2 (384 dims) — mesmo do embed_server.
Roda no Mac local (CPU/MPS). Demora ~20-40 min para 70K publicações.

Uso:
  python3 embeddings/generate_embeddings.py
  python3 embeddings/generate_embeddings.py --entity publications
  python3 embeddings/generate_embeddings.py --entity patents
  python3 embeddings/generate_embeddings.py --entity opportunities
  python3 embeddings/generate_embeddings.py --batch-size 128
"""

import argparse
import os
import sys
import time

import psycopg2
from psycopg2.extras import execute_values

DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "postgres://agora:agora_dev@localhost:5433/agora?sslmode=disable",
)
MODEL_NAME  = os.getenv("EMBED_MODEL", "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2")
BATCH_SIZE  = 64


def load_model():
    try:
        from sentence_transformers import SentenceTransformer
    except ImportError:
        print("Instalando sentence-transformers...")
        os.system(f"{sys.executable} -m pip install sentence-transformers -q")
        from sentence_transformers import SentenceTransformer
    print(f"[embeddings] Carregando modelo {MODEL_NAME}...")
    model = SentenceTransformer(MODEL_NAME)
    dims = model.get_sentence_embedding_dimension()
    print(f"[embeddings] Modelo pronto — {dims} dims")
    return model


def embed_publications(conn, model, batch_size: int):
    cur = conn.cursor()
    cur.execute("""
        SELECT id, title, COALESCE(abstract, '') AS abstract
        FROM publications
        WHERE embedding IS NULL
        ORDER BY id
    """)
    rows = cur.fetchall()
    total = len(rows)
    print(f"[embeddings] {total} publicações sem embedding")
    if total == 0:
        return

    _embed_batch(conn, model, rows, "publications", batch_size, total)


def embed_patents(conn, model, batch_size: int):
    cur = conn.cursor()
    cur.execute("""
        SELECT id, COALESCE(title,'') AS title, COALESCE(abstract,'') AS abstract
        FROM patents
        WHERE embedding IS NULL
        ORDER BY id
    """)
    rows = cur.fetchall()
    total = len(rows)
    print(f"[embeddings] {total} patentes sem embedding")
    if total == 0:
        return

    _embed_batch(conn, model, rows, "patents", batch_size, total)


def embed_opportunities(conn, model, batch_size: int):
    cur = conn.cursor()
    cur.execute("""
        SELECT id, title, COALESCE(description,'') AS description
        FROM opportunities
        WHERE embedding IS NULL
        ORDER BY id
    """)
    rows = cur.fetchall()
    total = len(rows)
    print(f"[embeddings] {total} oportunidades sem embedding")
    if total == 0:
        return

    _embed_batch(conn, model, rows, "opportunities", batch_size, total)


def _embed_batch(conn, model, rows, table: str, batch_size: int, total: int):
    cur = conn.cursor()
    processed = 0
    t0 = time.time()

    for i in range(0, total, batch_size):
        batch = rows[i : i + batch_size]
        texts = [f"{r[1]}. {r[2]}" for r in batch]
        embeddings = model.encode(texts, show_progress_bar=False, normalize_embeddings=True)

        updates = [(emb.tolist(), row[0]) for row, emb in zip(batch, embeddings)]
        cur.executemany(
            f"UPDATE {table} SET embedding = %s::vector WHERE id = %s",
            updates,
        )
        conn.commit()
        processed += len(batch)

        elapsed = time.time() - t0
        rate = processed / elapsed if elapsed > 0 else 0
        remaining = (total - processed) / rate if rate > 0 else 0
        print(
            f"  [{table}] {processed}/{total}  "
            f"({rate:.0f} reg/s, ~{remaining/60:.1f} min restantes)"
        )

    print(f"[embeddings] {table} concluído: {total} registros em {time.time()-t0:.0f}s")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--entity",
        choices=["publications", "patents", "opportunities", "all"],
        default="all",
    )
    parser.add_argument("--batch-size", type=int, default=BATCH_SIZE)
    args = parser.parse_args()

    model = load_model()
    conn  = psycopg2.connect(DATABASE_URL)

    try:
        if args.entity in ("publications", "all"):
            embed_publications(conn, model, args.batch_size)
        if args.entity in ("patents", "all"):
            embed_patents(conn, model, args.batch_size)
        if args.entity in ("opportunities", "all"):
            embed_opportunities(conn, model, args.batch_size)
    finally:
        conn.close()

    print("[embeddings] Tudo pronto!")


if __name__ == "__main__":
    main()
