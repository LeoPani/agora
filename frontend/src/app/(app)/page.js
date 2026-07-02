"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import {
  BookOpen, FlaskConical, Users, FileText,
  TrendingUp, Database, Activity, Radar
} from "lucide-react";
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import Link from "next/link";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

function KpiCard({ icon: Icon, label, value, sub, color }) {
  return (
    <div
      className="rounded-xl p-5 flex flex-col gap-3 transition-transform hover:-translate-y-[2px]"
      style={{ background: "var(--surface)", border: "1px solid var(--border)", boxShadow: "0 4px 20px rgba(0,0,0,0.15)" }}
    >
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold uppercase tracking-widest" style={{ color: "var(--text-muted)" }}>
          {label}
        </span>
        <div className="p-2 rounded-lg" style={{ background: color ? color + "20" : "var(--surface-2)" }}>
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
  const [signals, setSignals] = useState([]);
  const [loading, setLoad] = useState(true);

  useEffect(() => {
    Promise.all([
      fetch(`${API}/api/v1/stats`).then((r) => r.json()).catch(() => null),
      fetch(`${API}/api/v1/collector-runs`).then((r) => r.json()).catch(() => []),
      fetch(`${API}/api/v1/signals?limit=5`).then((r) => r.json()).catch(() => [])
    ]).then(([s, r, sig]) => {
      setStats(s);
      setRuns(Array.isArray(r) ? r.slice(0, 5) : []);
      setSignals(Array.isArray(sig) ? sig.slice(0, 5) : []);
      setLoad(false);
    });
  }, []);

  const fmt = (n) => (n != null ? Number(n).toLocaleString("pt-BR") : "—");

  const kpis = [
    { icon: BookOpen,     label: "Publicações",    value: fmt(stats?.publications),    sub: "OpenAlex + LOCUS",        color: "#A78BFA" },
    { icon: Users,        label: "Pesquisadores",   value: fmt(stats?.researchers),      sub: "com Lattes / ORCID",      color: "#34D399" },
    { icon: FlaskConical, label: "Patentes",        value: fmt(stats?.patents),          sub: "INPI + Lens.org",         color: "#F59E0B" },
    { icon: FileText,     label: "Oportunidades",  value: fmt(stats?.opportunities),    sub: "FAPEMIG · FINEP · CNPq",  color: "#FB7185" },
    { icon: Radar,        label: "Sinais Ativos",   value: fmt(stats?.active_signals),   sub: "Sinais não revisados",    color: "#D4A017" },
    { icon: TrendingUp,   label: "Tendências",      value: fmt(stats?.market_trends),    sub: "Google Trends (5 anos)",  color: "#C084FC" },
    { icon: Users,        label: "Grupos CNPq",     value: fmt(stats?.research_groups),  sub: "via DGP",                 color: "#60A5FA" },
    { icon: Database,     label: "Última Coleta",   value: stats?.last_collected ? new Date(stats.last_collected).toLocaleDateString('pt-BR') : "—", sub: "Status das coletas", color: "#9CA3AF" },
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

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* LineChart */}
        <div className="rounded-xl p-5 flex flex-col" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
          <div className="mb-4">
            <h2 className="text-sm font-semibold text-white">Publicações por Ano (Desde 2010)</h2>
          </div>
          <div className="flex-1 min-h-[250px]">
            {stats?.publications_by_year ? (
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={stats.publications_by_year}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#ffffff10" vertical={false} />
                  <XAxis dataKey="year" stroke="#ffffff50" fontSize={12} tickMargin={10} />
                  <YAxis stroke="#ffffff50" fontSize={12} tickFormatter={n => n > 1000 ? (n/1000).toFixed(1)+'k' : n} />
                  <Tooltip 
                    contentStyle={{ backgroundColor: "#1A1329", borderColor: "#6B21A8", borderRadius: "8px" }}
                    itemStyle={{ color: "#D4A017" }}
                  />
                  <Line type="monotone" dataKey="count" stroke="#A78BFA" strokeWidth={3} dot={false} activeDot={{ r: 6, fill: "#D4A017" }} />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <div className="w-full h-full flex items-center justify-center text-sm" style={{ color: "var(--text-muted)" }}>
                Sem dados ou carregando...
              </div>
            )}
          </div>
        </div>

        {/* Últimos Sinais */}
        <div className="rounded-xl p-5" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
          <div className="flex items-center gap-2 mb-4">
            <Radar size={16} style={{ color: "var(--accent-hover)" }} />
            <h2 className="text-sm font-semibold text-white">Últimos Sinais Detectados</h2>
          </div>
          {signals.length === 0 && !loading ? (
             <p className="text-sm py-4 text-center" style={{ color: "var(--text-muted)" }}>Nenhum sinal.</p>
          ) : (
            <div className="space-y-3">
              {signals.map(s => (
                <div key={s.id} className="p-3 rounded-lg flex flex-col gap-1" style={{ background: "var(--surface-2)" }}>
                  <div className="flex justify-between items-center">
                     <span className="text-xs uppercase tracking-wider text-[#D4A017] font-semibold">{s.signal_type}</span>
                     <Badge variant={s.relevance === "Alta" ? "success" : s.relevance === "Média" ? "gold" : "muted"}>
                       {s.relevance || "Alta"}
                     </Badge>
                  </div>
                  <h3 className="text-sm text-white font-medium line-clamp-1">{s.title}</h3>
                </div>
              ))}
              <div className="pt-2 text-right">
                <Link href="/signals" className="text-xs" style={{ color: "var(--accent-hover)" }}>Ver todos os sinais →</Link>
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Últimas coletas */}
        <div
          className="rounded-xl p-5"
          style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
        >
          <div className="flex items-center gap-2 mb-4">
            <Activity size={16} style={{ color: "var(--accent-hover)" }} />
            <h2 className="text-sm font-semibold text-white">Status dos Coletores</h2>
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
              <Link href="/collectors" className="text-xs" style={{ color: "var(--accent-hover)" }}>
                Ver histórico completo →
              </Link>
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
                  {["Fonte", "Camada", "Status"].map((h) => (
                    <th key={h} className="text-left pb-2 pr-6 text-xs font-semibold uppercase tracking-wider"
                      style={{ color: "var(--text-muted)" }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {[
                  ["OpenAlex (UFV)",    "Publicações",  "ok"],
                  ["LOCUS DSpace",      "Publicações",  "ok"],
                  ["INPI 775K",         "Patentes",     "ok"],
                  ["Google Patents",    "Patentes",     "ok"],
                  ["DGP/CNPq",          "Grupos",       "ok"],
                  ["Lens.org",          "Patentes",     "manual"],
                  ["Editais (4 fontes)","Oportunidades","ok"],
                  ["Comex Stat",        "Mercado",      "ok"],
                  ["Google Trends",     "Mercado",      "ok"],
                ].map(([fonte, camada, status]) => (
                  <tr key={fonte} style={{ borderBottom: "1px solid var(--border)" }}>
                    <td className="py-2.5 pr-6 text-white font-medium">{fonte}</td>
                    <td className="py-2.5 pr-6" style={{ color: "var(--text-muted)" }}>{camada}</td>
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

    </div>
  );
}
