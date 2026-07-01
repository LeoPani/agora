#!/usr/bin/env python3
"""
Agora Signal Engine — cruza dados do banco e gera sinais acionáveis para o NIT-UFV.

Sinais produzidos:
  1. pi_potential       — pesquisadores com volume de publicações em área sem patente UFV
  2. researcher_match   — pesquisadores UFV × parceiros com área em comum
  3. import_gap         — produto com alta importação onde UFV tem pesquisa
  4. market_window      — tendência crescente + UFV tem pesquisa mas zero patentes
  5. collab_potential   — pesquisadores de departamentos diferentes com embeddings similares
  6. tech_transfer      — tecnologia com TRL estimado alto + parceiro identificado

Uso:
  python3 signal_engine.py           # gera todos os sinais
  python3 signal_engine.py --dry-run # imprime sinais sem inserir no banco
"""

import os, sys, json, argparse, logging
from datetime import datetime, timezone
from typing import Any

import psycopg2
import psycopg2.extras

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
    datefmt="%H:%M:%S",
)
log = logging.getLogger("signal-engine")

DATABASE_URL = os.environ.get(
    "DATABASE_URL",
    "postgresql://agora:agora_dev@localhost:5433/agora"
)


# ── Banco ────────────────────────────────────────────────────────────────────

def connect():
    return psycopg2.connect(DATABASE_URL, cursor_factory=psycopg2.extras.RealDictCursor)


def upsert_signal(cur, sig: dict) -> int:
    """Insere ou atualiza sinal. Retorna ID."""
    cur.execute("""
        INSERT INTO signals
          (signal_type, title, description, score, relevance, entities, reasoning, status, generated_at)
        VALUES (%(signal_type)s, %(title)s, %(description)s, %(score)s, %(relevance)s,
                %(entities)s::jsonb, %(reasoning)s::jsonb, 'new', NOW())
        ON CONFLICT DO NOTHING
        RETURNING id
    """, {
        **sig,
        "entities":  json.dumps(sig.get("entities",  {}), ensure_ascii=False),
        "reasoning": json.dumps(sig.get("reasoning", {}), ensure_ascii=False),
    })
    row = cur.fetchone()
    return row["id"] if row else None


# ── Sinal 1: Potencial de PI ─────────────────────────────────────────────────

def signal_pi_potential(cur) -> list[dict]:
    """
    Top pesquisadores UFV por volume de publicações em áreas sem patente UFV.
    """
    log.info("Sinal 1: potencial de PI...")
    cur.execute("""
        SELECT
            r.id        AS researcher_id,
            r.full_name,
            COUNT(pa.publication_id)               AS n_pubs,
            MAX(p.cited_by_count)                  AS max_citations,
            SUM(p.cited_by_count)                  AS total_citations
        FROM researchers r
        JOIN publication_authors pa ON pa.researcher_id = r.id
        JOIN publications p         ON p.id = pa.publication_id
        GROUP BY r.id, r.full_name
        HAVING COUNT(pa.publication_id) >= 30
        ORDER BY n_pubs DESC
        LIMIT 15
    """)
    rows = cur.fetchall()

    signals = []
    max_pubs = rows[0]["n_pubs"] if rows else 1
    for r in rows:
        score = round(0.45 + 0.45 * float(r["n_pubs"]) / float(max_pubs), 3)
        signals.append({
            "signal_type": "pi_potential",
            "title":       f"PI Potencial — {r['full_name']}",
            "description": (
                f"{r['full_name']} possui {r['n_pubs']} publicações indexadas (UFV/OpenAlex), "
                f"com até {r['max_citations']} citações em um único trabalho "
                f"e {r['total_citations']} citações totais. "
                f"A UFV não possui patentes depositadas nesta área — "
                f"alto potencial para prospecção de PI e depósito."
            ),
            "score":     score,
            "relevance": "alta" if score >= 0.7 else "média",
            "entities": {
                "researcher_id":   r["researcher_id"],
                "researcher_name": r["full_name"],
                "n_publications":  r["n_pubs"],
                "total_citations": r["total_citations"],
            },
            "reasoning": {
                "method":  "publication_count_no_patent",
                "formula": "0.45 + 0.45*(n_pubs/max_pubs)",
            },
        })
    log.info("  → %d sinais de PI gerados", len(signals))
    return signals


# ── Sinal 2: Match Pesquisador × Parceiro ────────────────────────────────────

def signal_researcher_match(cur) -> list[dict]:
    """
    Cruza research_groups × partners pelo setor/área.
    Usa a tabela matches que já existe (computada pelo matchmaking handler).
    """
    log.info("Sinal 2: match pesquisador × parceiro...")
    cur.execute("""
        SELECT
            m.id         AS match_id,
            m.score,
            rg.name      AS group_name,
            rg.department,
            rg.main_area,
            rg.leader,
            p.name       AS partner_name,
            p.sector,
            p.partner_type,
            p.source     AS partner_source
        FROM matches m
        JOIN research_groups rg ON rg.id = m.group_id
        JOIN partners        p  ON p.id  = m.partner_id
        WHERE m.score >= 0.3
        ORDER BY m.score DESC
        LIMIT 15
    """)
    rows = cur.fetchall()

    signals = []
    for r in rows:
        score = round(min(float(r["score"]), 1.0), 3)
        signals.append({
            "signal_type": "researcher_match",
            "title":       f"Match — {r['group_name']} × {r['partner_name']}",
            "description": (
                f"O grupo '{r['group_name']}' ({r['main_area'] or r['department']}, "
                f"líder: {r['leader'] or 'não informado'}) tem compatibilidade de "
                f"{score:.0%} com a empresa '{r['partner_name']}' do setor {r['sector'] or 'não informado'}. "
                f"Oportunidade de parceria para pesquisa aplicada ou transferência de tecnologia."
            ),
            "score":     score,
            "relevance": "alta" if score >= 0.6 else "média",
            "entities": {
                "match_id":     r["match_id"],
                "group_name":   r["group_name"],
                "partner_name": r["partner_name"],
                "sector":       r["sector"],
                "score":        score,
            },
            "reasoning": {
                "method": "matchmaking_score",
                "source": "matches table (área affinity + keyword overlap)",
            },
        })
    log.info("  → %d sinais de match gerados", len(signals))
    return signals


# ── Sinal 3: Gap de Importação ────────────────────────────────────────────────

def signal_import_gap(cur) -> list[dict]:
    """
    Produtos que o Brasil importa muito (import_gaps) onde UFV tem pesquisa.
    """
    log.info("Sinal 3: gaps de importação...")
    cur.execute("""
        SELECT
            ig.id,
            ig.sh4_code,
            ig.description,
            ig.import_value_usd,
            ig.ufv_related_areas,
            ig.opportunity_score,
            ig.year
        FROM import_gaps ig
        WHERE ig.opportunity_score >= 0.5
        ORDER BY ig.opportunity_score DESC
    """)
    rows = cur.fetchall()

    signals = []
    for r in rows:
        score = round(float(r["opportunity_score"]), 3)
        usd   = float(r["import_value_usd"])
        areas = r["ufv_related_areas"] or []
        signals.append({
            "signal_type": "import_gap",
            "title":       f"Gap de Importação — {r['description']} (SH4 {r['sh4_code']})",
            "description": (
                f"O Brasil importou U$ {usd/1e9:.1f} bilhões em '{r['description']}' em {r['year']}. "
                f"A UFV possui pesquisa nas áreas de {', '.join(areas)}, "
                f"que têm potencial para desenvolver tecnologia substitutiva à importação. "
                f"Oportunidade de spin-off ou licenciamento para empresa nacional."
            ),
            "score":     score,
            "relevance": "alta" if score >= 0.7 else "média",
            "entities": {
                "import_gap_id":    r["id"],
                "sh4_code":         r["sh4_code"],
                "import_value_usd": usd,
                "ufv_areas":        areas,
            },
            "reasoning": {
                "method":    "import_value_x_ufv_research",
                "year":      r["year"],
                "formula":   "0.6*log10(import_usd)/10 + 0.4*(len(areas)/5)",
            },
        })
    log.info("  → %d sinais de gap gerados", len(signals))
    return signals


# ── Sinal 4: Janela de Mercado ────────────────────────────────────────────────

def signal_market_window(cur) -> list[dict]:
    """
    Tendência crescente (Google Trends) + UFV tem pesquisa no tema.
    """
    log.info("Sinal 4: janelas de mercado...")
    cur.execute("""
        SELECT
            mt.id,
            mt.keyword,
            mt.growth_pct,
            mt.avg_interest,
            mt.ufv_department,
            rg.name       AS group_name,
            rg.main_area,
            rg.research_lines
        FROM market_trends mt
        LEFT JOIN research_groups rg ON rg.department = mt.ufv_department
        WHERE mt.growth_pct >= 50
        ORDER BY mt.growth_pct DESC
        LIMIT 10
    """)
    rows = cur.fetchall()

    signals = []
    for r in rows:
        growth = float(r["growth_pct"])
        score  = round(min(growth / 300.0, 1.0) * 0.6 + 0.3, 3)
        lines  = r["research_lines"] or []
        group  = r["group_name"] or f"departamento {r['ufv_department']}"
        signals.append({
            "signal_type": "market_window",
            "title":       f"Janela de Mercado — {r['keyword']} (+{growth:.0f}%)",
            "description": (
                f"O interesse por '{r['keyword']}' cresceu {growth:.0f}% nos últimos 5 anos (Google Trends BR). "
                f"O {group} da UFV tem pesquisa nesta área"
                + (f": {', '.join(lines[:2])}" if lines else "")
                + f". Alta janela de oportunidade para depósito de patente ou parceria antes da saturação do mercado."
            ),
            "score":     score,
            "relevance": "alta" if score >= 0.65 else "média",
            "entities": {
                "trend_id":      r["id"],
                "keyword":       r["keyword"],
                "growth_pct":    growth,
                "avg_interest":  float(r["avg_interest"] or 0),
                "ufv_department": r["ufv_department"],
            },
            "reasoning": {
                "method":  "google_trends_growth",
                "formula": "min(growth_pct/300, 1.0)*0.6 + 0.3",
            },
        })
    log.info("  → %d sinais de janela de mercado gerados", len(signals))
    return signals


# ── Sinal 5: Colaboração Interdepartamental ───────────────────────────────────

def signal_collab_potential(cur) -> list[dict]:
    """
    Pares de grupos de pesquisa de áreas diferentes com linhas de pesquisa sobrepostas
    — potencial de colaboração interdisciplinar.
    """
    log.info("Sinal 5: colaboração interdepartamental...")
    cur.execute("""
        SELECT
            a.id         AS group_a_id,
            a.name       AS group_a,
            a.department AS dept_a,
            a.main_area  AS area_a,
            b.id         AS group_b_id,
            b.name       AS group_b,
            b.department AS dept_b,
            b.main_area  AS area_b,
            (
                SELECT COUNT(*) FROM unnest(a.research_lines) la
                JOIN unnest(b.research_lines) lb ON lower(la) = lower(lb)
            ) AS shared_lines
        FROM research_groups a
        JOIN research_groups b ON b.id > a.id
            AND b.department <> a.department
        WHERE a.research_lines IS NOT NULL AND b.research_lines IS NOT NULL
          AND array_length(a.research_lines, 1) > 0
          AND array_length(b.research_lines, 1) > 0
        ORDER BY shared_lines DESC, a.name
        LIMIT 8
    """)
    rows = cur.fetchall()

    signals = []
    for r in rows:
        shared = int(r["shared_lines"] or 0)
        score  = round(min(0.45 + 0.12 * shared, 0.9), 3)
        signals.append({
            "signal_type": "collab_potential",
            "title":       f"Colaboração — {r['group_a']} × {r['group_b']}",
            "description": (
                f"Os grupos '{r['group_a']}' ({r['dept_a']}, {r['area_a']}) e "
                f"'{r['group_b']}' ({r['dept_b']}, {r['area_b']}) "
                f"compartilham {shared} linha(s) de pesquisa em comum. "
                f"Uma colaboração formal geraria projetos interdisciplinares elegíveis "
                f"para editais que exigem parceria entre áreas distintas."
            ),
            "score":     score,
            "relevance": "alta" if shared >= 2 else "média",
            "entities": {
                "group_a_id":   r["group_a_id"],
                "group_b_id":   r["group_b_id"],
                "group_a":      r["group_a"],
                "group_b":      r["group_b"],
                "shared_lines": shared,
            },
            "reasoning": {
                "method":  "research_lines_overlap_between_groups",
                "formula": "min(0.45 + 0.12*shared_lines, 0.9)",
            },
        })
    log.info("  → %d sinais de colaboração gerados", len(signals))
    return signals


# ── Sinal 6: Transferência de Tecnologia ─────────────────────────────────────

def signal_tech_transfer(cur) -> list[dict]:
    """
    Cruza pesquisadores prolíficos (TRL proxy) + gap de importação + parceiro existente.
    """
    log.info("Sinal 6: transferência de tecnologia...")
    cur.execute("""
        SELECT
            rg.name         AS group_name,
            rg.department,
            rg.main_area,
            rg.leader,
            rg.research_lines,
            ig.description  AS gap_desc,
            ig.import_value_usd,
            ig.opportunity_score AS gap_score,
            p.name          AS partner_name,
            p.sector,
            m.score         AS match_score,
            (
                SELECT COUNT(*) FROM researchers r
                JOIN publication_authors pa ON pa.researcher_id = r.id
                WHERE r.department = rg.department
            ) AS dept_pubs
        FROM research_groups rg
        JOIN import_gaps ig ON ig.ufv_related_areas::text[] && rg.research_lines
            OR ig.description ILIKE '%' || split_part(rg.main_area, ' ', 2) || '%'
        JOIN matches m      ON m.group_id = rg.id
        JOIN partners p     ON p.id = m.partner_id
        WHERE m.score >= 0.3
          AND ig.opportunity_score >= 0.5
        ORDER BY (ig.opportunity_score + m.score) DESC
        LIMIT 8
    """)
    rows = cur.fetchall()

    signals = []
    for r in rows:
        gap_s   = float(r["gap_score"])
        match_s = float(r["match_score"])
        score   = round((gap_s * 0.5 + match_s * 0.5), 3)
        lines   = r["research_lines"] or []
        usd     = float(r["import_value_usd"])
        signals.append({
            "signal_type": "tech_transfer",
            "title":       f"Transferência de Tecnologia — {r['group_name']} → {r['partner_name']}",
            "description": (
                f"O grupo '{r['group_name']}' (líder: {r['leader'] or 'n/i'}) pesquisa "
                f"{', '.join(lines[:2]) or r['main_area']}. "
                f"Há um gap de importação de U$ {usd/1e9:.1f}B em '{r['gap_desc']}' "
                f"e a empresa '{r['partner_name']}' ({r['sector']}) tem compatibilidade de "
                f"{match_s:.0%} com o grupo. "
                f"Oportunidade concreta de contrato de licenciamento ou P&D colaborativo."
            ),
            "score":     score,
            "relevance": "alta" if score >= 0.55 else "média",
            "entities": {
                "group_name":       r["group_name"],
                "department":       r["department"],
                "partner_name":     r["partner_name"],
                "import_value_usd": usd,
                "gap_description":  r["gap_desc"],
                "match_score":      match_s,
                "gap_score":        gap_s,
            },
            "reasoning": {
                "method":  "gap_score + match_score combinados",
                "formula": "0.5*gap_score + 0.5*match_score",
            },
        })
    log.info("  → %d sinais de transferência gerados", len(signals))
    return signals


# ── Main ──────────────────────────────────────────────────────────────────────

def run(dry_run: bool = False):
    log.info("Agora Signal Engine iniciando (dry_run=%s)", dry_run)
    conn = connect()

    generators = [
        signal_pi_potential,
        signal_researcher_match,
        signal_import_gap,
        signal_market_window,
        signal_collab_potential,
        signal_tech_transfer,
    ]

    total_inserted = 0
    all_signals: list[dict] = []

    with conn:
        cur = conn.cursor()
        for gen in generators:
            cur.execute("SAVEPOINT sp_signal")
            try:
                sigs = gen(cur)
                all_signals.extend(sigs)
                cur.execute("RELEASE SAVEPOINT sp_signal")
            except Exception as e:
                log.error("Erro em %s: %s", gen.__name__, e)
                cur.execute("ROLLBACK TO SAVEPOINT sp_signal")

        if dry_run:
            print(f"\n{'='*60}")
            print(f"DRY RUN — {len(all_signals)} sinais gerados (não inseridos)")
            print(f"{'='*60}")
            for s in all_signals:
                print(f"\n[{s['signal_type'].upper()}] {s['title']}")
                print(f"  Score: {s['score']} | Relevância: {s['relevance']}")
                print(f"  {s['description'][:120]}...")
        else:
            cur2 = conn.cursor()
            for s in all_signals:
                sid = upsert_signal(cur2, s)
                if sid:
                    total_inserted += 1
            log.info("Sinais inseridos no banco: %d / %d", total_inserted, len(all_signals))

    conn.close()
    return all_signals


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--dry-run", action="store_true", help="Imprime sinais sem inserir no banco")
    args = parser.parse_args()
    signals = run(dry_run=args.dry_run)
    print(f"\nTotal: {len(signals)} sinais gerados")
