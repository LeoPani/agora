"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { BarChart3, Search } from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

function ScoreBar({ value }) {
  const pct = Math.min(Math.round((value || 0) * 100), 100);
  const color = pct >= 70 ? "#34D399" : pct >= 40 ? "#FBBF24" : "#A78BFA";
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-1.5 rounded-full" style={{ background: "var(--surface-2)" }}>
        <div className="h-full rounded-full transition-all" style={{ width: `${pct}%`, background: color }} />
      </div>
      <span className="text-xs tabular-nums w-8 text-right" style={{ color }}>{pct}</span>
    </div>
  );
}

export default function ImportsPage() {
  const [gaps, setGaps]     = useState([]);
  const [loading, setLoad]  = useState(true);
  const [q, setQ]           = useState("");

  useEffect(() => {
    fetch(`${API}/api/v1/import-gaps?limit=200`)
      .then((r) => r.json())
      .catch(() => [])
      .then((d) => { setGaps(Array.isArray(d) ? d : d?.import_gaps ?? []); setLoad(false); });
  }, []);

  const filtered = gaps
    .filter((g) => !q ||
      (g.description || "").toLowerCase().includes(q.toLowerCase()) ||
      (g.sh4_code || "").includes(q))
    .sort((a, b) => (b.opportunity_score || 0) - (a.opportunity_score || 0));

  const totalUSD = gaps.reduce((s, g) => s + (g.import_value_usd || 0), 0);

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Gaps de Importação</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Produtos importados onde UFV tem capacidade de substituição tecnológica (Comex Stat 2023)
        </p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
        {[
          { label: "Total de gaps", value: gaps.length, color: "#A78BFA" },
          { label: "Valor total importado (USD)", value: totalUSD >= 1e9 ? `US$ ${(totalUSD/1e9).toFixed(1)}B` : `US$ ${(totalUSD/1e6).toFixed(0)}M`, color: "#F59E0B" },
          { label: "Categorias SH4", value: new Set(gaps.map((g) => g.sh4_code)).size, color: "#34D399" },
        ].map(({ label, value, color }) => (
          <div key={label} className="rounded-xl p-5" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
            <p className="text-xs font-semibold uppercase tracking-widest mb-2" style={{ color: "var(--text-muted)" }}>{label}</p>
            <p className="text-2xl font-bold" style={{ color }}>{typeof value === "number" ? value.toLocaleString("pt-BR") : value}</p>
          </div>
        ))}
      </div>

      <div className="relative">
        <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2" style={{ color: "var(--text-muted)" }} />
        <input
          className="w-full pl-9 pr-4 py-2 text-sm rounded-lg"
          style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
          placeholder="Buscar por produto ou código SH4…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>

      <div className="rounded-xl overflow-hidden" style={{ border: "1px solid var(--border)" }}>
        <table className="w-full text-sm">
          <thead>
            <tr style={{ background: "var(--surface)" }}>
              {["SH4", "Produto", "País Origem", "Importado (USD)", "Áreas UFV", "Score"].map((h) => (
                <th key={h} className="text-left px-4 py-3 text-xs font-semibold uppercase tracking-wider"
                  style={{ color: "var(--text-muted)", borderBottom: "1px solid var(--border)" }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr><td colSpan={6} className="text-center py-8" style={{ color: "var(--text-muted)" }}>Carregando…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={6} className="text-center py-8" style={{ color: "var(--text-muted)" }}>
                Nenhum gap encontrado. Execute <code>make collect-comex ingest-comex</code>.
              </td></tr>
            )}
            {filtered.slice(0, 100).map((g, i) => {
              const usd = g.import_value_usd || 0;
              const usdFmt = usd >= 1e6
                ? `US$ ${(usd / 1e6).toFixed(1)}M`
                : `US$ ${Math.round(usd).toLocaleString("pt-BR")}`;
              return (
                <tr key={g.id || i} style={{ borderBottom: "1px solid var(--border)" }}>
                  <td className="px-4 py-3 font-mono text-xs text-white">{g.sh4_code}</td>
                  <td className="px-4 py-3 text-white max-w-xs truncate">{g.description || "—"}</td>
                  <td className="px-4 py-3" style={{ color: "var(--text-muted)" }}>{g.country_origin || "—"}</td>
                  <td className="px-4 py-3 tabular-nums font-medium" style={{ color: "#FBBF24" }}>{usdFmt}</td>
                  <td className="px-4 py-3">
                    <div className="flex gap-1 flex-wrap">
                      {(g.ufv_related_areas || []).slice(0, 2).map((a) => (
                        <Badge key={a} variant="muted">{a}</Badge>
                      ))}
                      {(g.ufv_related_areas || []).length > 2 && (
                        <span className="text-xs" style={{ color: "var(--text-muted)" }}>+{g.ufv_related_areas.length - 2}</span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 w-28"><ScoreBar value={g.opportunity_score} /></td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
