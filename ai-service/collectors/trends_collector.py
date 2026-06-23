#!/usr/bin/env python3
"""
Google Trends Collector — Rastreia interesse em temas onde UFV tem pesquisa.

Usa pytrends (wrapper não-oficial da API do Google Trends).
Calcula crescimento % entre primeiros e últimos 12 meses do período.

Output: ai-service/data/market_trends.jsonl

Dependências: pip install pytrends
"""

import json, time, math
from pathlib import Path
from datetime import datetime

OUTPUT_DIR  = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
OUTPUT_FILE = OUTPUT_DIR / "market_trends.jsonl"

DELAY     = 3.0  # Google Trends tem rate limit informal
TIMEFRAME = "today 5-y"
GEO       = "BR"

# Termos de busca organizados por departamento UFV
KEYWORDS_BY_DEPT = {
    "DFT": [
        "melhoramento genético plantas",
        "soja resistente seca",
        "milho biotecnologia",
        "cultivar transgênica",
    ],
    "DTA": [
        "alimentos funcionais",
        "fermentados probióticos",
        "carne plant based",
        "queijo artesanal",
    ],
    "DFP": [
        "biocontrole pragas",
        "fungicida biológico",
        "defensivo orgânico",
        "biopesticida",
    ],
    "DPI": [
        "inteligência artificial agricultura",
        "precision agriculture",
        "drone agrícola",
        "visão computacional campo",
    ],
    "DBB": [
        "biofertilizante",
        "inoculante rizóbio",
        "microbioma solo",
        "CRISPR plantas",
    ],
    "DEF": [
        "eucalipto carbono",
        "manejo florestal",
        "biomassa energia",
        "reflorestamento produtivo",
    ],
    "DEA": [
        "irrigação inteligente",
        "mecanização agrícola",
        "sensoriamento remoto lavoura",
        "agricultura digital",
    ],
    "DQI": [
        "nanotecnologia agrícola",
        "bioproduto natural",
        "extrato vegetal bioativo",
        "polímero biodegradável",
    ],
}


def compute_growth(interest_series: list[int]) -> tuple[float, int, int]:
    """Retorna (growth_pct, avg_interest, peak_interest)."""
    if not interest_series or all(v == 0 for v in interest_series):
        return 0.0, 0, 0

    avg     = sum(interest_series) // len(interest_series)
    peak    = max(interest_series)
    n       = len(interest_series)
    quarter = max(1, n // 4)

    first_avg = sum(interest_series[:quarter]) / quarter
    last_avg  = sum(interest_series[-quarter:]) / quarter

    if first_avg == 0:
        growth = 100.0 if last_avg > 0 else 0.0
    else:
        growth = round((last_avg - first_avg) / first_avg * 100, 2)

    return growth, avg, peak


def collect_keyword(pt, keyword: str, dept: str) -> dict | None:
    """Coleta dados de uma keyword via pytrends."""
    try:
        pt.build_payload([keyword], cat=0, timeframe=TIMEFRAME, geo=GEO, gprop="")
        df = pt.interest_over_time()

        if df.empty or keyword not in df.columns:
            return None

        series       = df[keyword].tolist()
        growth, avg, peak = compute_growth(series)

        related_q = {}
        related_t = {}
        try:
            rq = pt.related_queries()
            if keyword in rq and rq[keyword].get("rising") is not None:
                top_q = rq[keyword]["rising"].head(5)
                related_q = top_q.to_dict("records")
        except Exception:
            pass

        try:
            rt = pt.related_topics()
            if keyword in rt and rt[keyword].get("rising") is not None:
                top_t = rt[keyword]["rising"].head(5)
                related_t = top_t[["topic_title", "value"]].to_dict("records")
        except Exception:
            pass

        return {
            "keyword":         keyword,
            "geo":             GEO,
            "timeframe":       TIMEFRAME,
            "avg_interest":    avg,
            "peak_interest":   peak,
            "growth_pct":      growth,
            "related_queries": related_q,
            "related_topics":  related_t,
            "ufv_department":  dept,
            "raw_data": {
                "series_length": len(series),
                "collected_at":  datetime.now().isoformat(),
            },
        }
    except Exception as e:
        print(f"    ERRO {keyword}: {e}")
        return None


def collect():
    try:
        from pytrends.request import TrendReq
    except ImportError:
        print("ERRO: instale: pip install pytrends")
        import sys; sys.exit(1)

    print(f"Coletando Google Trends (geo={GEO}, período={TIMEFRAME})...")
    pt = TrendReq(hl="pt-BR", tz=180, timeout=(10, 30), retries=2, backoff_factor=0.5)

    results = []
    total_kw = sum(len(v) for v in KEYWORDS_BY_DEPT.values())
    done = 0

    for dept, keywords in KEYWORDS_BY_DEPT.items():
        for kw in keywords:
            done += 1
            print(f"  [{done}/{total_kw}] {dept} — {kw}")
            rec = collect_keyword(pt, kw, dept)
            if rec:
                results.append(rec)
            time.sleep(DELAY)

    # Ordenar por crescimento
    results.sort(key=lambda x: x["growth_pct"], reverse=True)

    with open(OUTPUT_FILE, "w") as f:
        for r in results:
            f.write(json.dumps(r, ensure_ascii=False, default=str) + "\n")

    print(f"\nGoogle Trends: {len(results)} keywords coletadas")
    print(f"Top 3 crescimento:")
    for r in results[:3]:
        print(f"  {r['keyword']}: +{r['growth_pct']}%")


if __name__ == "__main__":
    collect()
