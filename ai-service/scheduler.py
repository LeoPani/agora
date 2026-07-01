#!/usr/bin/env python3
"""
Agora Python Scheduler — roda collectors Python periodicamente dentro do Docker.

Responsabilidades:
- Roda os collectors Python (openalex, trends, comex, editais, dgp...)
- Após cada coleta, chama POST /internal/ingest/:source na API Go para
  disparar o ingest correspondente (que roda o binário compilado)
- Registra execuções em collector_runs no Postgres

Env vars:
  DATABASE_URL   — conexão Postgres
  BACKEND_URL    — URL da API Go (default: http://api:8081)
  DATA_DIR       — onde salvar JSONLs (default: /app/data)
"""

import os, sys, subprocess, time, json, logging
from datetime import datetime, timezone
from pathlib import Path

import schedule
import requests
import psycopg2

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%SZ",
)
log = logging.getLogger("py-scheduler")

BACKEND_URL  = os.environ.get("BACKEND_URL", "http://api:8081")
DATABASE_URL = os.environ.get("DATABASE_URL", "")
DATA_DIR     = Path(os.environ.get("DATA_DIR", "/app/data"))
COLLECTORS   = Path(__file__).parent / "collectors"


# ── Postgres ────────────────────────────────────────────────────────────────

def db_conn():
    return psycopg2.connect(DATABASE_URL)


def log_run(source: str, status: str, records: int = 0, error: str = ""):
    if not DATABASE_URL:
        return
    try:
        with db_conn() as conn, conn.cursor() as cur:
            cur.execute("""
                INSERT INTO collector_runs (source, status, records_found, error, started_at, finished_at)
                VALUES (%s, %s, %s, %s, %s, %s)
                ON CONFLICT DO NOTHING
            """, (source, status, records, error,
                  datetime.now(timezone.utc), datetime.now(timezone.utc)))
    except Exception as e:
        log.warning("log_run failed: %s", e)


# ── Runner ──────────────────────────────────────────────────────────────────

def run_collector(name: str, script: str) -> bool:
    """Executa um script Python collector e retorna True se sucesso."""
    path = COLLECTORS / script
    if not path.exists():
        log.error("collector not found: %s", path)
        return False

    log.info("starting collector: %s", name)
    t0 = time.monotonic()
    try:
        result = subprocess.run(
            [sys.executable, str(path)],
            capture_output=True, text=True, timeout=3600,
            env={**os.environ, "DATA_DIR": str(DATA_DIR)},
        )
        elapsed = time.monotonic() - t0
        if result.returncode == 0:
            log.info("collector %s OK (%.0fs)", name, elapsed)
            log_run(name, "success")
            return True
        else:
            log.error("collector %s FAILED:\n%s", name, result.stderr[-2000:])
            log_run(name, "error", error=result.stderr[-500:])
            return False
    except subprocess.TimeoutExpired:
        log.error("collector %s TIMEOUT", name)
        log_run(name, "error", error="timeout")
        return False
    except Exception as e:
        log.error("collector %s exception: %s", name, e)
        log_run(name, "error", error=str(e))
        return False


def trigger_ingest(source: str):
    """Chama POST /internal/ingest/:source na API para disparar o ingest Go."""
    url = f"{BACKEND_URL}/internal/ingest/{source}"
    try:
        r = requests.post(url, timeout=600)
        if r.status_code == 200:
            log.info("ingest %s triggered OK", source)
        else:
            log.warning("ingest %s → HTTP %s", source, r.status_code)
    except Exception as e:
        log.warning("ingest trigger %s failed: %s", source, e)


def job(name: str, script: str, ingest_source: str | None = None):
    """Combina collector + trigger de ingest."""
    ok = run_collector(name, script)
    if ok and ingest_source:
        trigger_ingest(ingest_source)


# ── Agendamentos ─────────────────────────────────────────────────────────────

def setup_schedule():
    # OpenAlex — mensal
    schedule.every(30).days.do(job, "openalex", "openalex_collector.py", "openalex")

    # LOCUS — semanal
    schedule.every(7).days.do(job, "locus", "locus_collector.py", "locus")

    # DGP — mensal
    schedule.every(30).days.do(job, "dgp", "dgp_collector.py", "dgp")

    # Editais — semanal
    schedule.every(7).days.do(job, "fapemig",  "fapemig_collector.py",  None)
    schedule.every(7).days.do(job, "finep",     "finep_collector.py",    None)
    schedule.every(7).days.do(job, "cnpq",      "cnpq_collector.py",     None)
    schedule.every(7).days.do(job, "embrapii",  "embrapii_collector.py", None)
    schedule.every(7).days.do(trigger_ingest,   "opportunities")

    # Comex Stat — mensal
    schedule.every(30).days.do(job, "comex", "comex_collector.py", "comex")

    # Google Trends — quinzenal
    schedule.every(15).days.do(job, "trends", "trends_collector.py", "trends")


def run_all_now():
    """Executa todos os jobs imediatamente na primeira inicialização."""
    log.info("running initial collection pass...")
    for j in schedule.get_jobs():
        j.run()


def main():
    log.info("agora py-scheduler starting (backend=%s)", BACKEND_URL)
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    setup_schedule()

    # Roda primeira vez com pequeno delay para API subir
    time.sleep(30)
    # Roda apenas jobs leves na primeira vez (não openalex que é pesado)
    job("comex",   "comex_collector.py",   "comex")
    job("trends",  "trends_collector.py",  "trends")
    job("dgp",     "dgp_collector.py",     "dgp")

    log.info("entering scheduler loop")
    while True:
        schedule.run_pending()
        time.sleep(60)


if __name__ == "__main__":
    main()
