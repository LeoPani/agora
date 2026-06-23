#!/usr/bin/env python3
"""
Lens.org Patent Parser — Processa export CSV manual do Lens.org.

FLUXO MANUAL (necessário, pois Lens API é paga):
  1. Acesse https://www.lens.org/lens/search/patent/list
  2. Busque: applicant.name:"UNIVERSIDADE FEDERAL DE VICOSA"
  3. Clique em "Export" → CSV → máximo 1000 registros
  4. Salve como ~/Downloads/lens_export.csv
  5. Execute: make collect-lens

Output:
- ai-service/data/lens_patents.jsonl
- ai-service/data/lens_citations.jsonl

Dependências: pip install requests (para fuzzy matching futuro)
"""

import csv
import json
import sys
import re
import unicodedata
from pathlib import Path
from datetime import datetime

DEFAULT_INPUT = Path.home() / "Downloads" / "lens_export.csv"
OUTPUT_DIR    = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

# Mapeamento de colunas do Lens CSV para nosso schema
# O Lens exporta com nomes de coluna em inglês
LENS_COL_MAP = {
    "Lens ID":                    "lens_id",
    "Title":                      "title",
    "Abstract":                   "abstract",
    "Application Number":         "application_number",
    "Publication Number":         "publication_number",
    "Filing Date":                "filing_date",
    "Publication Date":           "publication_date",
    "Grant Date":                 "grant_date",
    "Applicants":                 "applicants",
    "Inventors":                  "inventors",
    "Jurisdiction":               "jurisdiction",
    "Legal Status":               "legal_status",
    "INPI Number":                "inpi_number",
    "International Classifications": "ipc_codes",
    "Patent Citations":           "patent_citations",
    "Non Patent Citations":       "npl_citations",
    "Cited By Patent Count":      "cited_by_count",
    "Family Size":                "family_size",
    "Source URLs":                "source_urls",
}

# Nomes alternativos de colunas Lens
ALT_COL_MAP = {
    "Lens Id":                    "lens_id",
    "Simple Family Size":         "family_size",
    "Legal Status (Simple)":      "legal_status",
    "IPC Classifications":        "ipc_codes",
    "Cited By":                   "cited_by_count",
}


def normalize_col(col: str) -> str:
    return LENS_COL_MAP.get(col) or ALT_COL_MAP.get(col) or col.lower().replace(" ", "_")


def parse_list_field(val: str) -> list[str]:
    if not val:
        return []
    # Lens separa com "; " ou ";"
    return [v.strip() for v in re.split(r";\s*", val) if v.strip()]


def parse_date(val: str) -> str | None:
    if not val:
        return None
    # Lens usa YYYY-MM-DD
    val = val.strip()
    if re.match(r"\d{4}-\d{2}-\d{2}", val):
        return val[:10]
    # Tenta DD/MM/YYYY
    m = re.match(r"(\d{2})/(\d{2})/(\d{4})", val)
    if m:
        return f"{m.group(3)}-{m.group(2)}-{m.group(1)}"
    return None


def extract_inpi_number(row: dict) -> str | None:
    """Tenta extrair o número INPI do registro Lens."""
    # Coluna direta
    if row.get("inpi_number"):
        return row["inpi_number"].strip()
    # Do application_number (formato BR...)
    app = row.get("application_number", "") or ""
    if app.startswith("BR"):
        return app.strip()
    # Do publication_number
    pub = row.get("publication_number", "") or ""
    if pub.startswith("BR"):
        return pub.strip()
    return None


def run(input_path: Path | None = None):
    path = input_path or DEFAULT_INPUT

    if not path.exists():
        print(f"ERRO: Arquivo não encontrado: {path}")
        print()
        print("Siga as instruções em:")
        print("  ai-service/collectors/lens_README.md")
        sys.exit(1)

    print(f"Lendo {path}...")

    patents = []
    citations = []

    with open(path, newline="", encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        if not reader.fieldnames:
            print("ERRO: CSV vazio ou sem cabeçalho.")
            sys.exit(1)

        # Normalizar nomes de colunas
        col_map = {col: normalize_col(col) for col in reader.fieldnames}
        print(f"  Colunas encontradas: {list(reader.fieldnames)[:6]}...")

        for i, raw_row in enumerate(reader):
            row = {col_map[k]: v for k, v in raw_row.items()}

            lens_id = row.get("lens_id", "").strip()
            title   = row.get("title", "").strip()
            if not lens_id and not title:
                continue

            inpi_number = extract_inpi_number(row)

            npl_list     = parse_list_field(row.get("npl_citations", ""))
            patent_cits  = parse_list_field(row.get("patent_citations", ""))
            ipc_list     = parse_list_field(row.get("ipc_codes", ""))
            inventors    = parse_list_field(row.get("inventors", ""))
            applicants   = parse_list_field(row.get("applicants", ""))

            # Família size
            try:
                family_size = int(row.get("family_size") or 0)
            except (ValueError, TypeError):
                family_size = 0

            try:
                cited_by = int(row.get("cited_by_count") or 0)
            except (ValueError, TypeError):
                cited_by = 0

            patent = {
                "lens_id":         lens_id,
                "inpi_number":     inpi_number,
                "title":           title,
                "abstract":        row.get("abstract", "").strip(),
                "application_number": row.get("application_number", "").strip(),
                "filing_date":     parse_date(row.get("filing_date")),
                "publication_date":parse_date(row.get("publication_date")),
                "grant_date":      parse_date(row.get("grant_date")),
                "applicants":      applicants,
                "inventors":       inventors,
                "jurisdiction":    row.get("jurisdiction", "").strip(),
                "legal_status":    row.get("legal_status", "").strip(),
                "ipc_codes":       ipc_list,
                "patent_citations":patent_cits,
                "npl_citations":   npl_list,
                "cited_by_count":  cited_by,
                "family_size":     family_size,
            }
            patents.append(patent)

            # Extrair citações NPL separadamente
            for npl in npl_list:
                if npl:
                    citations.append({
                        "lens_id":       lens_id,
                        "inpi_number":   inpi_number,
                        "npl_text":      npl,
                        "citation_type": "npl",
                    })
            for pat_cit in patent_cits:
                if pat_cit:
                    citations.append({
                        "lens_id":       lens_id,
                        "inpi_number":   inpi_number,
                        "npl_text":      pat_cit,
                        "citation_type": "patent",
                    })

    patents_file  = OUTPUT_DIR / "lens_patents.jsonl"
    citations_file = OUTPUT_DIR / "lens_citations.jsonl"

    with open(patents_file, "w") as f:
        for p in patents:
            f.write(json.dumps(p, ensure_ascii=False) + "\n")

    with open(citations_file, "w") as f:
        for c in citations:
            f.write(json.dumps(c, ensure_ascii=False) + "\n")

    print(f"\nLens parser finalizado:")
    print(f"  {len(patents)} patentes")
    print(f"  {len(citations)} citações (NPL + patentes)")
    print(f"  Saída: {patents_file}")
    print(f"  Saída: {citations_file}")
    print(f"\nPróximo: make ingest-lens")


if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=Path, default=None)
    args = parser.parse_args()
    run(args.input)
