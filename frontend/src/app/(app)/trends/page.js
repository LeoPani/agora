"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { TrendingUp, Search } from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

function MiniBar({ value, peak }) {
  if (!peak || peak === 0) return <span style={{ color: "var(--text-muted)" }}>—</span>;
  const pct = Math.round((value / peak) * 100);
  const color = value >= 60 ? "#34D399" : value >= 30 ? "#FBBF24" : "#A78BFA";
  return (
    <div className="flex items-center gap-2">
      <div className="w-16 h-1.5 rounded-full" style={{ background: "var(--surface-2)" }}>
        <div className="h-full rounded-full" style={{ width: `${pct}%`, background: color }} />
      </div>
      <span className="text-xs tabular-nums" style={{ color }}>{value}</span>
    </div>
  );
}

function GrowthBadge({ value }) {
  if (value == null) return null;
  const pos = value > 0;
  return (
    <span className="text-xs font-bold tabular-nums"
      style={{ color: pos ? "#34D399" : value < -10 ? "#FB7185" : "var(--text-muted)" }}>
      {pos ? "+" : ""}{value.toFixed(1)}%
    </span>
  );
}

const DEPT_LABELS = {
  DFT: "Fitotecnia",  DTA: "Tec. Alimentos", DFP: "Fitopatologia",
  DPI: "Informática", DBB: "Bioquímica",      DEF: "Engenharia Florestal",
  DEA: "Eng. Agrícola", DQI: "Química",
};

export default function TrendsPage() {
  const [trends, setTrends] = useState([]);
  const [loading, setLoad]  = useState(true);
  const [q, setQ]           = useState("");
  const [dept, setDept]     = useState("all");

  useEffect(() => {
    fetch(`${API}/api/v1/trends?limit=200`)
      .then((r) => r.json())
      .catch(() => [])
      .then((d) => { setTrends(Array.isArray(d) ? d : d?.trends ?? []); setLoad(false); });
  }, []);

  const depts = ["all", ...new Set(trends.map((t) => t.ufv_department).filter(Boolean))];

  const filtered = trends
    .filter((t) => {
      const matchQ = !q || (t.keyword || "").toLowerCase().includes(q.toLowerCase());
      const matchD = dept === "all" || t.ufv_department === dept;
      return matchQ && matchD;
    })
    .sort((a, b) => (b.growth_pct || 0) - (a.growth_pct || 0));

  const topGrowth = filtered.slice(0, 3);

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Tendências de Mercado</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Google Trends — 5 anos — Brasil — 32 keywords × 8 departamentos UFV
        </p>
      </div>

      {/* Top 3 crescimento */}
      {topGrowth.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {topGrowth.map((t) => (
            <div key={t.keyword} className="rounded-xl p-5"
              style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
              <div className="flex items-center justify-between mb-2">
                <Badge variant="muted">{t.ufv_department ? DEPT_LABELS[t.ufv_department] || t.ufv_department : "—"}</Badge>
                <GrowthBadge value={t.growth_pct} />
              </div>
              <p className="text-sm font-semibold text-white">{t.keyword}</p>
              <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>
                Pico: {t.peak_interest} · Média: {t.avg_interest}
              </p>
            </div>
          ))}
        </div>
      )}

      {/* Filtros */}
      <div className="flex gap-3 items-center flex-wrap">
        <div className="relative flex-1 min-w-48">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2" style={{ color: "var(--text-muted)" }} />
          <input
            className="w-full pl-9 pr-4 py-2 text-sm rounded-lg"
            style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
            placeholder="Buscar keyword…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
        <select
          className="px-3 py-2 text-sm rounded-lg"
          style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
          value={dept}
          onChange={(e) => setDept(e.target.value)}
        >
          {depts.map((d) => <option key={d} value={d}>
            {d === "all" ? "Todos os departamentos" : `${d} — ${DEPT_LABELS[d] || d}`}
          </option>)}
        </select>
      </div>

      {/* Tabela */}
      <div className="rounded-xl overflow-hidden" style={{ border: "1px solid var(--border)" }}>
        <table className="w-full text-sm">
          <thead>
            <tr style={{ background: "var(--surface)" }}>
              {["Keyword", "Departamento", "Crescimento (5a)", "Interesse médio", "Pico"].map((h) => (
                <th key={h} className="text-left px-4 py-3 text-xs font-semibold uppercase tracking-wider"
                  style={{ color: "var(--text-muted)", borderBottom: "1px solid var(--border)" }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr><td colSpan={5} className="text-center py-8" style={{ color: "var(--text-muted)" }}>Carregando…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={5} className="text-center py-8" style={{ color: "var(--text-muted)" }}>
                Nenhuma tendência. Execute <code>make collect-trends ingest-trends</code>.
              </td></tr>
            )}
            {filtered.map((t, i) => (
              <tr key={t.id || t.keyword + i} style={{ borderBottom: "1px solid var(--border)" }}>
                <td className="px-4 py-3 text-white font-medium">{t.keyword}</td>
                <td className="px-4 py-3">
                  {t.ufv_department && (
                    <Badge variant="muted">{t.ufv_department}</Badge>
                  )}
                </td>
                <td className="px-4 py-3"><GrowthBadge value={t.growth_pct} /></td>
                <td className="px-4 py-3">
                  <MiniBar value={t.avg_interest || 0} peak={t.peak_interest || 0} />
                </td>
                <td className="px-4 py-3 tabular-nums" style={{ color: "var(--text-muted)" }}>
                  {t.peak_interest ?? "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
