#!/usr/bin/env python3
"""
Comex Stat Collector — Coleta gaps de importação do MDIC/Comex Stat.

Identifica produtos que o Brasil importa muito mas UFV tem pesquisa —
ou seja, oportunidades onde a tecnologia local pode substituir importação.

API: https://api-comexstat.mdic.gov.br/general
Output: ai-service/data/comex_import_gaps.jsonl

Dependências: pip install requests
"""

import requests, json, time
from pathlib import Path
from datetime import datetime

OUTPUT_DIR  = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
OUTPUT_FILE = OUTPUT_DIR / "comex_import_gaps.jsonl"

API_BASE = "https://api-comexstat.mdic.gov.br/general"
DELAY    = 1.5
HEADERS  = {
    "User-Agent": "Agora/1.0 (UFV NIT)",
    "Accept":     "application/json",
}

# Ano de referência para o cálculo de gaps
REFERENCE_YEAR = 2023

# SH4 de interesse agrário/biotec com alta importação
# Baseado em setores onde UFV tem força pesquisadora
SH4_CATEGORIES = {
    # Insumos agrícolas
    "3101": "Adubos/fertilizantes orgânicos",
    "3102": "Adubos nitrogenados minerais",
    "3105": "Adubos minerais mixtos",
    "2933": "Compostos orgânicos (piridina/quinolina)",
    # Defensivos agrícolas
    "3808": "Pesticidas e herbicidas",
    "3002": "Vacinas e produtos imunológicos",
    # Biotecnologia / equipamentos lab
    "3826": "Biodiesel",
    "3004": "Medicamentos (dose)",
    "3006": "Produtos farmacêuticos especializados",
    # Alimentos processados com potencial
    "1901": "Preparados alimentares de cereais",
    "2106": "Preparações alimentícias diversas",
    "0306": "Crustáceos",
    # Tecnologia / máquinas agrícolas
    "8432": "Máquinas preparação solo",
    "8433": "Colhedoras agrícolas",
    "8437": "Máquinas beneficiamento cereais",
}

# Mapeamento SH4 → áreas UFV com pesquisa relevante
UFV_RESEARCH_MAP = {
    "3101": ["Solos", "Fitotecnia"],
    "3102": ["Solos", "Nutrição Mineral de Plantas"],
    "3105": ["Solos", "Fitotecnia"],
    "3808": ["Fitopatologia", "Entomologia"],
    "3002": ["Veterinária", "Microbiologia"],
    "3826": ["Agroenergia", "Biotecnologia"],
    "3004": ["Química", "Bioquímica"],
    "1901": ["Tecnologia de Alimentos"],
    "2106": ["Tecnologia de Alimentos", "Nutrição"],
    "8432": ["Engenharia Agrícola", "Mecanização"],
    "8433": ["Engenharia Agrícola", "Mecanização"],
    "8437": ["Engenharia Agrícola"],
}


def fetch_imports_for_sh4(sh4: str, year: int) -> dict | None:
    """Busca importações para um produto SH4 em um ano."""
    params = {
        "flow": "import",
        "monthStart": f"{year}-01",
        "monthEnd":   f"{year}-12",
        "sh4":        sh4,
        "groupBy":    "sh4,country",
        "details":    "true",
    }
    try:
        r = requests.get(API_BASE, params=params, headers=HEADERS, timeout=20)
        if r.status_code == 200:
            return r.json()
        print(f"  SH4 {sh4}: HTTP {r.status_code}")
    except Exception as e:
        print(f"  SH4 {sh4}: {e}")
    return None


def compute_opportunity_score(value_usd: float, ufv_areas: list[str]) -> float:
    """Score simples: maior importação + mais áreas UFV = maior oportunidade."""
    if value_usd <= 0:
        return 0.0
    import math
    log_val  = min(math.log10(value_usd + 1) / 10.0, 1.0)
    area_fac = min(len(ufv_areas) / 5.0, 1.0)
    return round(0.6 * log_val + 0.4 * area_fac, 4)


def collect():
    print(f"Coletando gaps de importação Comex Stat (ano {REFERENCE_YEAR})...")
    gaps = []

    for sh4, desc in SH4_CATEGORIES.items():
        print(f"  SH4 {sh4} — {desc}")
        data = fetch_imports_for_sh4(sh4, REFERENCE_YEAR)
        time.sleep(DELAY)

        if not data:
            continue

        # Estrutura da API: {"data": {"list": [...]}}
        records = (data.get("data", {}).get("list", [])
                   or data.get("list", [])
                   or (data if isinstance(data, list) else []))

        if not records:
            # Fallback: sumarizar apenas o total
            records = [data] if isinstance(data, dict) else []

        for rec in records:
            value   = float(rec.get("metricFOB")     or rec.get("value_usd")     or 0)
            kg      = float(rec.get("metricKG")       or rec.get("kg")            or 0)
            country = rec.get("noCountry")            or rec.get("country")        or "N/A"

            ufv_areas = UFV_RESEARCH_MAP.get(sh4, [])
            score     = compute_opportunity_score(value, ufv_areas)

            gaps.append({
                "sh4_code":          sh4,
                "description":       desc,
                "country_origin":    str(country),
                "year":              REFERENCE_YEAR,
                "import_value_usd":  value,
                "import_kg":         kg,
                "ufv_related_areas": ufv_areas,
                "opportunity_score": score,
                "raw_data":          rec,
            })

    # Ordenar por score descendente e pegar top 200
    gaps.sort(key=lambda x: x["opportunity_score"], reverse=True)
    top_gaps = gaps[:200]

    with open(OUTPUT_FILE, "w") as f:
        for g in top_gaps:
            f.write(json.dumps(g, ensure_ascii=False, default=str) + "\n")

    print(f"\nComex Stat: {len(top_gaps)} gaps salvos (de {len(gaps)} total)")


if __name__ == "__main__":
    collect()
