"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import {
  BookOpen, FlaskConical, Users, FileText,
  BarChart3, TrendingUp, Database, Activity,
} from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

function KpiCard({ icon: Icon, label, value, sub, color }) {
  return (
    <div
      className="rounded-xl p-5 flex flex-col gap-3"
      style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
    >
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold uppercase tracking-widest" style={{ color: "var(--text-muted)" }}>
          {label}
        </span>
        <div className="p-2 rounded-lg" style={{ background: "var(--surface-2)" }}>
          <Icon size={16} style={{ color: color || "var(--accent-hover)" }} />
        </div>
      </div>
      <div>
        <p className="text-3xl font-bold text-white">{value ?? "—"}</p>
        {sub && <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>{sub}</p>}
      </div>
    </div>
  );
}

function RunRow({ run }) {
  const variant = run.status === "ok" ? "success" : run.status === "running" ? "gold" : "error";
  const date = run.started_at ? new Date(run.started_at).toLocaleString("pt-BR") : "—";
  return (
    <div className="flex items-center gap-3 py-2.5" style={{ borderBottom: "1px solid var(--border)" }}>
      <Badge variant={variant}>{run.status}</Badge>
      <span className="flex-1 text-sm text-white truncate">{run.collector_name}</span>
      <span className="text-xs tabular-nums" style={{ color: "var(--text-muted)" }}>
        {(run.records_collected ?? 0).toLocaleString("pt-BR")} reg
      </span>
      <span className="text-xs" style={{ color: "var(--text-muted)", minWidth: "10rem", textAlign: "right" }}>
        {date}
      </span>
    </div>
  );
}

export default function HomePage() {
  const [stats, setStats]  = useState(null);
  const [runs, setRuns]    = useState([]);
  const [loading, setLoad] = useState(true);

  useEffect(() => {
    Promise.all([
      fetch(`${API}/api/v1/stats`).then((r) => r.json()).catch(() => null),
      fetch(`${API}/api/v1/collector-runs`).then((r) => r.json()).catch(() => []),
    ]).then(([s, r]) => {
      setStats(s);
      setRuns(Array.isArray(r) ? r.slice(0, 8) : []);
      setLoad(false);
    });
  }, []);

  const fmt = (n) => (n != null ? Number(n).toLocaleString("pt-BR") : "—");

  const kpis = [
    { icon: BookOpen,     label: "Publicações",    value: fmt(stats?.publications),    sub: "OpenAlex + LOCUS",        color: "#A78BFA" },
    { icon: FlaskConical, label: "Patentes",        value: fmt(stats?.patents),          sub: "INPI + Lens.org",         color: "#F59E0B" },
    { icon: Users,        label: "Pesquisadores",   value: fmt(stats?.researchers),      sub: "com Lattes / ORCID",      color: "#34D399" },
    { icon: Users,        label: "Grupos CNPq",     value: fmt(stats?.research_groups),  sub: "via DGP",                 color: "#60A5FA" },
    { icon: FileText,     label: "Editais Ativos",  value: fmt(stats?.opportunities),    sub: "FAPEMIG · FINEP · CNPq",  color: "#FB7185" },
    { icon: BarChart3,    label: "Gaps Importação", value: fmt(stats?.import_gaps),      sub: "Comex Stat 2023",         color: "#FBBF24" },
    { icon: TrendingUp,   label: "Tendências",      value: fmt(stats?.market_trends),    sub: "Google Trends (5 anos)",  color: "#C084FC" },
    { icon: Database,     label: "Coletas",         value: fmt(stats?.collector_runs),   sub: "Total de runs",           color: "var(--text-muted)" },
  ];

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-white">Visão Geral</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Radar de Inteligência de Inovação · NIT-UFV
        </p>
      </div>

      {/* 8 KPI cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {kpis.map((k) => <KpiCard key={k.label} {...k} />)}
      </div>

      {/* Últimas coletas */}
      <div
        className="rounded-xl p-5"
        style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
      >
        <div className="flex items-center gap-2 mb-4">
          <Activity size={16} style={{ color: "var(--accent-hover)" }} />
          <h2 className="text-sm font-semibold text-white">Últimas Coletas</h2>
        </div>
        {loading && (
          <p className="text-sm py-4 text-center" style={{ color: "var(--text-muted)" }}>Carregando…</p>
        )}
        {!loading && runs.length === 0 && (
          <p className="text-sm py-4 text-center" style={{ color: "var(--text-muted)" }}>
            Nenhuma coleta registrada. Execute <code>make collect-all ingest-all</code>.
          </p>
        )}
        {runs.map((r) => <RunRow key={r.id} run={r} />)}
        {runs.length > 0 && (
          <div className="pt-3">
            <a href="/collectors" className="text-xs" style={{ color: "var(--accent-hover)" }}>
              Ver histórico completo →
            </a>
          </div>
        )}
      </div>

      {/* Fontes */}
      <div
        className="rounded-xl p-5"
        style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
      >
        <h2 className="text-sm font-semibold text-white mb-4">Fontes de Dados</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr style={{ borderBottom: "1px solid var(--border)" }}>
                {["Fonte", "Camada", "Volume estimado", "Frequência", "Status"].map((h) => (
                  <th key={h} className="text-left pb-2 pr-6 text-xs font-semibold uppercase tracking-wider"
                    style={{ color: "var(--text-muted)" }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {[
                ["OpenAlex (UFV)",    "Publicações",  "~47K works",     "Mensal",     "ok"],
                ["LOCUS DSpace",      "Publicações",  "~25K teses",     "Semanal",    "ok"],
                ["INPI 775K",         "Patentes",     "775K pedidos BR","Semestral",  "ok"],
                ["Google Patents",    "Patentes",     "~235 UFV",       "Mensal",     "ok"],
                ["DGP/CNPq",          "Grupos",       "~120 grupos",    "Mensal",     "ok"],
                ["Lens.org",          "Patentes",     "CSV manual",     "Mensal",     "manual"],
                ["Editais (4 fontes)","Oportunidades","variável",        "Semanal",   "ok"],
                ["Comex Stat",        "Mercado",      "200 gaps SH4",   "Mensal",     "ok"],
                ["Google Trends",     "Mercado",      "32 keywords",    "Quinzenal",  "ok"],
              ].map(([fonte, camada, vol, freq, status]) => (
                <tr key={fonte} style={{ borderBottom: "1px solid var(--border)" }}>
                  <td className="py-2.5 pr-6 text-white font-medium">{fonte}</td>
                  <td className="py-2.5 pr-6" style={{ color: "var(--text-muted)" }}>{camada}</td>
                  <td className="py-2.5 pr-6 tabular-nums" style={{ color: "var(--text-muted)" }}>{vol}</td>
                  <td className="py-2.5 pr-6" style={{ color: "var(--text-muted)" }}>{freq}</td>
                  <td className="py-2.5">
                    <Badge variant={status === "ok" ? "success" : status === "manual" ? "gold" : "muted"}>
                      {status}
                    </Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
