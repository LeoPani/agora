#!/usr/bin/env python3
"""
INPI Dataset Loader — Carrega LeoPani/inpi-patent-abstracts do HuggingFace
e salva em JSONL chunked (50K por arquivo) para ingestão Go.

Output:
- ai-service/data/inpi_patents_part_001.jsonl
- ai-service/data/inpi_patents_part_002.jsonl
- ...  (total ~16 arquivos para 775K registros)

Dependências: pip install datasets
"""

import json
import sys
from pathlib import Path

OUTPUT_DIR = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

DATASET_ID = "LeoPani/inpi-patent-abstracts"
CHUNK_SIZE  = 50_000

# Colunas esperadas no dataset INPI
COL_MAP = {
    "numero_pedido":     "inpi_number",
    "titulo":            "title",
    "resumo":            "abstract",
    "classificacao_ipc": "ipc_code",
    "secao_ipc":         "ipc_section",
    "data_deposito":     "filing_date",
    "depositante":       "applicant",
    "tipo_depositante":  "applicant_type",
}

UFV_TOKENS = ["VICOSA", "VIÇOSA", "UFV", "UNIVERSIDADE FEDERAL DE VICOSA"]


def is_ufv(applicant: str) -> bool:
    up = (applicant or "").upper()
    return any(tok in up for tok in UFV_TOKENS)


def to_record(row: dict) -> dict:
    rec = {}
    for src, dst in COL_MAP.items():
        val = row.get(src)
        if val is None:
            val = row.get(dst)  # tenta o nome destino também
        rec[dst] = val
    rec["is_ufv"] = is_ufv(rec.get("applicant", "") or "")
    rec["raw_data"] = {k: v for k, v in row.items() if k not in COL_MAP}
    return rec


def run():
    try:
        from datasets import load_dataset
    except ImportError:
        print("ERRO: instale a dependência: pip install datasets")
        sys.exit(1)

    print(f"Carregando {DATASET_ID} do HuggingFace...")
    ds = load_dataset(DATASET_ID, split="train")
    total = len(ds)
    print(f"  {total:,} registros carregados")

    chunk_idx = 1
    buffer = []
    ufv_count = 0
    written = 0

    def flush(buf: list, idx: int):
        path = OUTPUT_DIR / f"inpi_patents_part_{idx:03d}.jsonl"
        with open(path, "w") as f:
            for rec in buf:
                f.write(json.dumps(rec, ensure_ascii=False, default=str) + "\n")
        print(f"  Salvo: {path.name} ({len(buf):,} registros)")

    for i, row in enumerate(ds):
        rec = to_record(dict(row))
        buffer.append(rec)
        if rec["is_ufv"]:
            ufv_count += 1

        if len(buffer) >= CHUNK_SIZE:
            flush(buffer, chunk_idx)
            written += len(buffer)
            chunk_idx += 1
            buffer = []

        if (i + 1) % 100_000 == 0:
            print(f"  Processado: {i+1:,}/{total:,} | UFV: {ufv_count}")

    if buffer:
        flush(buffer, chunk_idx)
        written += len(buffer)

    print(f"\nINPI dataset carregado:")
    print(f"  {written:,} registros totais")
    print(f"  {ufv_count} registros UFV identificados")
    print(f"  {chunk_idx} arquivo(s) gerados")
    print(f"\nPróximo passo: make ingest-inpi")


if __name__ == "__main__":
    run()
