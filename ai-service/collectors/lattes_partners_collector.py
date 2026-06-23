#!/usr/bin/env python3
"""
Lattes Partners Collector — Encontra pesquisadores externos com interesse nas áreas UFV.

Estratégia:
1. Busca no portal de busca textual do Lattes (buscatextual.cnpq.br)
   por pesquisadores que trabalham nos mesmos temas das patentes UFV
2. Filtra por vínculo institucional ≠ UFV (pesquisadores em empresas/outras IES)
3. Gera perfis com link Lattes, área, vínculo atual

Output: ai-service/data/partners_lattes.jsonl

Dependências: requests, beautifulsoup4
"""

import requests, json, time, re, unicodedata
from pathlib import Path
from bs4 import BeautifulSoup

OUTPUT_DIR  = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
OUTPUT_FILE = OUTPUT_DIR / "partners_lattes.jsonl"

DELAY   = 3.0
HEADERS = {
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
    "Accept-Language": "pt-BR,pt;q=0.9",
}

BASE_URL = "https://buscatextual.cnpq.br/buscatextual/busca.do"

# Termos de busca por área — cada um gera uma página de resultados
SEARCH_TERMS = [
    ("melhoramento genético plantas",      "Melhoramento/Fitotecnia"),
    ("biocontrole fitopatógenos",          "Fitopatologia"),
    ("alimentos funcionais probióticos",   "Tecnologia de Alimentos"),
    ("biofertilizante inoculante",         "Microbiologia/Solos"),
    ("agricultura de precisão drone",      "Eng. Agrícola"),
    ("eucalipto celulose biomassa",        "Eng. Florestal"),
    ("nanotecnologia agrícola",            "Química/Nano"),
    ("inteligência artificial agropecuária","IA/Agro"),
    ("vacina veterinária imunológico",     "Veterinária"),
    ("biopesticida controle biológico",    "Entomologia"),
]

# Instituições a excluir (própria UFV e afiliadas)
UFV_TERMS = ["universidade federal de viçosa", "ufv", "vicosa"]


def normalize(s: str) -> str:
    s = unicodedata.normalize("NFD", s.lower())
    return "".join(c for c in s if unicodedata.category(c) != "Mn")


def is_ufv(institution: str) -> bool:
    norm = normalize(institution)
    return any(t in norm for t in UFV_TERMS)


def search_lattes(term: str, max_results: int = 20) -> list[dict]:
    """Busca pesquisadores no Lattes por termo."""
    params = {
        "metodo":        "forwardPaginaResultados",
        "registros":     f"0;{max_results}",
        "query":         term,
        "tipoConsulta":  "RESGATARTERMOS",
        "tipoBusca":     "B",
        "textoBusca":    term,
        "buscarDemaisDocumentos": "1",
    }
    try:
        r = requests.get(BASE_URL, params=params, headers=HEADERS, timeout=25)
        if r.status_code != 200:
            print(f"  Lattes {term[:30]}: HTTP {r.status_code}")
            return []
        return parse_results(r.text, term)
    except Exception as e:
        print(f"  Lattes {term[:30]}: {e}")
        return []


def parse_results(html: str, search_term: str) -> list[dict]:
    """Extrai pesquisadores da página de resultados do Lattes."""
    soup    = BeautifulSoup(html, "html.parser")
    results = []

    # Lattes retorna cards com nome, instituição e link para currículo
    items = (soup.select(".resultado-busca, .item-resultado, li.resultado")
             or soup.select("div.resultado"))

    if not items:
        # Fallback: qualquer link que aponte para curriculo.cnpq.br
        links = soup.select("a[href*='curriculo.cnpq.br'], a[href*='lattes.cnpq.br/']")
        for a in links[:20]:
            href = a.get("href", "")
            name = a.get_text(strip=True)
            if name and len(name) > 3:
                results.append({
                    "name":         name,
                    "lattes_url":   href,
                    "institution":  "",
                    "area":         search_term,
                })
        return results

    for item in items[:20]:
        name_el  = item.select_one(".nome, h3, h4, strong, .titulo-resultado")
        inst_el  = item.select_one(".instituicao, .afiliacao, .vinculo")
        link_el  = item.select_one("a[href*='curriculo'], a[href*='lattes']")

        name     = name_el.get_text(strip=True) if name_el else ""
        inst     = inst_el.get_text(strip=True) if inst_el else ""
        lattes_url = link_el["href"] if link_el else ""

        if not name or is_ufv(inst):
            continue

        results.append({
            "name":        name,
            "lattes_url":  lattes_url,
            "institution": inst,
            "area":        search_term,
        })

    return results


def collect():
    print("Coletando pesquisadores externos via Lattes/CNPq...")

    all_partners = {}  # nome normalizado → dict

    for term, area_label in SEARCH_TERMS:
        print(f"  Buscando: {term[:40]}")
        results = search_lattes(term)
        print(f"    → {len(results)} encontrados")

        for r in results:
            norm = normalize(r["name"])
            if norm in all_partners:
                # Acumula áreas
                all_partners[norm]["ufv_areas"] = list(set(
                    all_partners[norm].get("ufv_areas", []) + [area_label]
                ))
                all_partners[norm]["interest_score"] = min(
                    all_partners[norm]["interest_score"] + 0.1, 1.0
                )
            else:
                all_partners[norm] = {
                    "name":            r["name"],
                    "normalized_name": norm,
                    "partner_type":    "pesquisador",
                    "sector":          "Academia/Empresa",
                    "location":        r["institution"],
                    "lattes_url":      r["lattes_url"],
                    "lattes_id":       extract_lattes_id(r["lattes_url"]),
                    "ufv_areas":       [area_label],
                    "interest_score":  0.4,
                    "source":          "lattes",
                    "raw_data":        r,
                }
        time.sleep(DELAY)

    partners = list(all_partners.values())
    # Pesquisadores com mais áreas têm score maior
    for p in partners:
        p["interest_score"] = min(0.3 + len(p["ufv_areas"]) * 0.15, 1.0)

    with open(OUTPUT_FILE, "w") as f:
        for p in partners:
            f.write(json.dumps(p, ensure_ascii=False, default=str) + "\n")

    print(f"\nLattes Partners: {len(partners)} pesquisadores salvos")


def extract_lattes_id(url: str) -> str | None:
    m = re.search(r"(\d{16})", url or "")
    return m.group(1) if m else None


if __name__ == "__main__":
    collect()
