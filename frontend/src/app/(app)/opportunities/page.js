"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { FileText, Search, Clock } from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

const SOURCE_COLORS = {
  FAPEMIG: "#34D399",
  FINEP:   "#60A5FA",
  CNPq:    "#A78BFA",
  EMBRAPII:"#F59E0B",
};

function daysUntil(deadline) {
  if (!deadline) return null;
  const d = new Date(deadline);
  if (isNaN(d)) return null;
  const diff = Math.ceil((d - Date.now()) / 86400000);
  return diff;
}

function OpportunityCard({ op }) {
  const color  = SOURCE_COLORS[op.source] || "var(--text-muted)";
  const days   = daysUntil(op.deadline);
  const urgent = days !== null && days <= 7 && days >= 0;
  const expired = days !== null && days < 0;

  return (
    <div
      className="rounded-xl p-5 flex flex-col gap-3"
      style={{
        background: "var(--surface)",
        border: `1px solid ${urgent ? "#FB718540" : "var(--border)"}`,
      }}
    >
      <div className="flex items-start justify-between gap-2">
        <span className="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded"
          style={{ background: color + "20", color }}>
          {op.source}
        </span>
        {days !== null && (
          <span className="flex items-center gap-1 text-xs" style={{ color: expired ? "#FB7185" : urgent ? "#FBBF24" : "var(--text-muted)" }}>
            <Clock size={11} />
            {expired ? "Encerrado" : days === 0 ? "Hoje" : `${days}d`}
          </span>
        )}
      </div>
      <h3 className="text-sm font-semibold text-white leading-snug line-clamp-2">{op.title}</h3>
      {op.description && (
        <p className="text-xs leading-relaxed line-clamp-3" style={{ color: "var(--text-muted)" }}>
          {op.description}
        </p>
      )}
      <div className="flex items-center gap-2 mt-auto pt-1">
        <Badge variant={expired ? "error" : urgent ? "gold" : op.status === "aberto" ? "success" : "muted"}>
          {expired ? "encerrado" : op.status || "aberto"}
        </Badge>
        {op.url && (
          <a href={op.url} target="_blank" rel="noreferrer"
            className="text-xs ml-auto" style={{ color: "var(--accent-hover)" }}>
            Ver edital ↗
          </a>
        )}
      </div>
    </div>
  );
}

export default function OpportunitiesPage() {
  const [opps, setOpps]      = useState([]);
  const [loading, setLoad]   = useState(true);
  const [q, setQ]            = useState("");
  const [src, setSrc]        = useState("all");

  useEffect(() => {
    fetch(`${API}/api/v1/opportunities?limit=200`)
      .then((r) => r.json())
      .catch(() => [])
      .then((d) => { setOpps(Array.isArray(d) ? d : d?.opportunities ?? []); setLoad(false); });
  }, []);

  const sources = ["all", ...new Set(opps.map((o) => o.source))];
  const filtered = opps.filter((o) => {
    const matchQ = !q || (o.title || "").toLowerCase().includes(q.toLowerCase());
    const matchS = src === "all" || o.source === src;
    return matchQ && matchS;
  }).sort((a, b) => {
    const da = daysUntil(a.deadline) ?? 9999;
    const db = daysUntil(b.deadline) ?? 9999;
    return da - db;
  });

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Oportunidades</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Editais e chamadas públicas — FAPEMIG · FINEP · CNPq · EMBRAPII
        </p>
      </div>

      {/* KPIs por fonte */}
      <div className="flex gap-4 flex-wrap">
        {Object.entries(SOURCE_COLORS).map(([s, color]) => {
          const cnt = opps.filter((o) => o.source === s).length;
          return (
            <div key={s} className="rounded-xl px-5 py-3 flex gap-3 items-center"
              style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
              <span className="w-2 h-2 rounded-full" style={{ background: color }} />
              <span className="text-sm font-medium text-white">{s}</span>
              <span className="text-xl font-bold text-white">{cnt}</span>
            </div>
          );
        })}
      </div>

      {/* Filtros */}
      <div className="flex gap-3 items-center flex-wrap">
        <div className="relative flex-1 min-w-48">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2" style={{ color: "var(--text-muted)" }} />
          <input
            className="w-full pl-9 pr-4 py-2 text-sm rounded-lg"
            style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
            placeholder="Buscar edital…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
        <select
          className="px-3 py-2 text-sm rounded-lg"
          style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
          value={src}
          onChange={(e) => setSrc(e.target.value)}
        >
          {sources.map((s) => <option key={s} value={s}>{s === "all" ? "Todas as fontes" : s}</option>)}
        </select>
      </div>

      {loading && (
        <div className="py-16 text-center" style={{ color: "var(--text-muted)" }}>Carregando…</div>
      )}
      {!loading && filtered.length === 0 && (
        <div className="py-16 text-center" style={{ color: "var(--text-muted)" }}>
          Nenhum edital. Execute <code>make collect-editais ingest-opportunities</code>.
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filtered.map((o, i) => <OpportunityCard key={o.id || o.external_id || i} op={o} />)}
      </div>
    </div>
  );
}
