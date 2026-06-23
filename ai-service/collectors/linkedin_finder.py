#!/usr/bin/env python3
"""
LinkedIn Finder — Gera leads para busca manual no LinkedIn.

LinkedIn não permite scraping (ToS) nem tem API gratuita para busca de empresas.
Esta ferramenta gera:
1. URLs de busca pré-formatadas para LinkedIn Company Search
2. Queries para LinkedIn People Search
3. Um CSV exportável para prospecção manual pelo NIT

Uso: python3 linkedin_finder.py [--open]
  --open: abre as URLs no browser automaticamente

Output: ai-service/data/linkedin_search_leads.json
"""

import json, sys, unicodedata, re
from pathlib import Path
from urllib.parse import quote

OUTPUT_DIR  = Path(__file__).parent.parent / "data"
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
OUTPUT_FILE = OUTPUT_DIR / "linkedin_search_leads.json"

# Setores LinkedIn → áreas UFV correspondentes
LINKEDIN_SECTORS = [
    {
        "sector":         "Agricultural",
        "ufv_areas":      ["Fitotecnia", "Melhoramento", "Zootecnia"],
        "keywords":       ["agricultural biotechnology", "plant breeding", "agtech"],
        "description":    "Empresas de biotecnologia agrícola e melhoramento de plantas",
    },
    {
        "sector":         "Biotechnology Research",
        "ufv_areas":      ["Biotecnologia", "Bioquímica", "Microbiologia"],
        "keywords":       ["biotech", "biopesticide", "biofertilizer"],
        "description":    "Startups e empresas de biotech aplicada ao agro",
    },
    {
        "sector":         "Food and Beverage Services",
        "ufv_areas":      ["Tecnologia de Alimentos"],
        "keywords":       ["functional food", "food innovation", "food tech"],
        "description":    "Indústrias alimentícias com interesse em inovação",
    },
    {
        "sector":         "Paper & Forest Products",
        "ufv_areas":      ["Eng. Florestal"],
        "keywords":       ["eucalyptus", "cellulose", "forestry management"],
        "description":    "Empresas do setor florestal e celulose",
    },
    {
        "sector":         "Veterinary",
        "ufv_areas":      ["Veterinária", "Zootecnia"],
        "keywords":       ["veterinary pharmaceutical", "animal health", "livestock"],
        "description":    "Farmacêuticas veterinárias e saúde animal",
    },
    {
        "sector":         "Environmental Services",
        "ufv_areas":      ["Eng. Ambiental", "Solos", "Recursos Hídricos"],
        "keywords":       ["environmental technology", "soil remediation", "water treatment"],
        "description":    "Empresas de tecnologia ambiental",
    },
    {
        "sector":         "Software Development",
        "ufv_areas":      ["IA", "Informática"],
        "keywords":       ["precision agriculture software", "agri-tech platform", "farm management"],
        "description":    "Empresas de software para agronegócio",
    },
]

# Perfis de pesquisadores/executivos para busca no LinkedIn People
PEOPLE_QUERIES = [
    {"title": "P&D",         "keywords": ["pesquisa desenvolvimento agronegócio", "agricultural research manager"]},
    {"title": "Inovação",    "keywords": ["gerente inovação agroindústria", "innovation director food"]},
    {"title": "Licenciamento","keywords": ["technology licensing agriculture", "IP manager biotech"]},
    {"title": "Biotech CTO", "keywords": ["CTO biotecnologia agrícola", "chief technology officer agtech"]},
]


def build_company_search_url(keyword: str, sector: str = "") -> str:
    base = "https://www.linkedin.com/search/results/companies/"
    q = quote(keyword)
    return f"{base}?keywords={q}&origin=GLOBAL_SEARCH_HEADER"


def build_people_search_url(keyword: str, title: str = "") -> str:
    base = "https://www.linkedin.com/search/results/people/"
    q    = quote(keyword)
    t    = quote(title) if title else ""
    url  = f"{base}?keywords={q}&origin=GLOBAL_SEARCH_HEADER"
    if title:
        url += f"&title={t}"
    return url


def build_google_proxy_url(query: str) -> str:
    """Busca Google que retorna perfis públicos do LinkedIn."""
    q = quote(f'site:linkedin.com/company {query} brasil')
    return f"https://www.google.com/search?q={q}"


def generate_leads() -> dict:
    company_leads = []
    for s in LINKEDIN_SECTORS:
        for kw in s["keywords"]:
            company_leads.append({
                "keyword":        kw,
                "sector":         s["sector"],
                "ufv_areas":      s["ufv_areas"],
                "description":    s["description"],
                "linkedin_url":   build_company_search_url(kw),
                "google_url":     build_google_proxy_url(kw),
                "type":           "company",
            })

    people_leads = []
    for p in PEOPLE_QUERIES:
        for kw in p["keywords"]:
            people_leads.append({
                "keyword":      kw,
                "title_filter": p["title"],
                "linkedin_url": build_people_search_url(kw, p["title"]),
                "type":         "person",
            })

    return {
        "generated_at":  __import__("datetime").datetime.now().isoformat(),
        "instructions":  (
            "Abra cada linkedin_url e exporte os resultados manualmente "
            "ou use o Sales Navigator se disponível. "
            "Para busca gratuita, use os google_url que retornam perfis públicos."
        ),
        "company_searches": company_leads,
        "people_searches":  people_leads,
        "total_queries":    len(company_leads) + len(people_leads),
    }


def run(open_browser: bool = False):
    print("Gerando leads LinkedIn para prospecção...")
    leads = generate_leads()

    with open(OUTPUT_FILE, "w") as f:
        json.dump(leads, f, ensure_ascii=False, indent=2)

    print(f"\nGerados {leads['total_queries']} queries:")
    print(f"  {len(leads['company_searches'])} buscas de empresas")
    print(f"  {len(leads['people_searches'])} buscas de pessoas")
    print(f"\nSalvo em: {OUTPUT_FILE}")

    if open_browser:
        import webbrowser
        for lead in leads["company_searches"][:3]:
            webbrowser.open(lead["linkedin_url"])
            print(f"  Abrindo: {lead['keyword']}")

    print("\nInstrução: Abra linkedin_search_leads.json no frontend /partners")
    print("para ver as queries organizadas por área UFV.")


if __name__ == "__main__":
    run(open_browser="--open" in sys.argv)
