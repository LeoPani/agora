"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Users, Search } from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

function GroupCard({ group }) {
  const areaColor = {
    "Ciências Agrárias":   "#34D399",
    "Ciências Biológicas": "#60A5FA",
    "Ciências Exatas":     "#A78BFA",
    "Engenharias":         "#F59E0B",
    "Ciências da Saúde":   "#FB7185",
  }[group.main_area] || "var(--text-muted)";

  return (
    <div className="rounded-xl p-5 flex flex-col gap-3"
      style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
      <div className="flex items-start justify-between gap-2">
        <h3 className="text-sm font-semibold text-white leading-snug">{group.name}</h3>
        {group.department && <Badge variant="muted">{group.department}</Badge>}
      </div>
      {group.main_area && (
        <div className="flex items-center gap-1.5">
          <span className="w-2 h-2 rounded-full shrink-0" style={{ background: areaColor }} />
          <span className="text-xs" style={{ color: "var(--text-muted)" }}>{group.main_area}</span>
        </div>
      )}
      {group.leader && (
        <p className="text-xs" style={{ color: "var(--text-muted)" }}>
          Líder: <span className="text-white">{group.leader}</span>
        </p>
      )}
      {(group.research_lines || []).length > 0 && (
        <div className="flex gap-1.5 flex-wrap">
          {group.research_lines.slice(0, 3).map((l) => (
            <span key={l} className="text-[10px] px-2 py-0.5 rounded-full"
              style={{ background: "var(--surface-2)", color: "var(--text-muted)", border: "1px solid var(--border)" }}>
              {l}
            </span>
          ))}
          {group.research_lines.length > 3 && (
            <span className="text-[10px]" style={{ color: "var(--text-muted)" }}>
              +{group.research_lines.length - 3}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

export default function GroupsPage() {
  const [groups, setGroups]  = useState([]);
  const [loading, setLoad]   = useState(true);
  const [q, setQ]            = useState("");
  const [area, setArea]      = useState("all");

  useEffect(() => {
    fetch(`${API}/api/v1/groups?limit=200`)
      .then((r) => r.json())
      .catch(() => [])
      .then((d) => { setGroups(Array.isArray(d) ? d : d?.groups ?? []); setLoad(false); });
  }, []);

  const areas = ["all", ...new Set(groups.map((g) => g.main_area).filter(Boolean))];

  const filtered = groups.filter((g) => {
    const matchQ = !q ||
      (g.name || "").toLowerCase().includes(q.toLowerCase()) ||
      (g.leader || "").toLowerCase().includes(q.toLowerCase());
    const matchA = area === "all" || g.main_area === area;
    return matchQ && matchA;
  });

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Grupos de Pesquisa</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Grupos UFV registrados no DGP/CNPq
        </p>
      </div>

      <div className="flex gap-3 items-center flex-wrap">
        <div className="relative flex-1 min-w-48">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2" style={{ color: "var(--text-muted)" }} />
          <input
            className="w-full pl-9 pr-4 py-2 text-sm rounded-lg"
            style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
            placeholder="Buscar por nome ou líder…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
        <select
          className="px-3 py-2 text-sm rounded-lg"
          style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
          value={area}
          onChange={(e) => setArea(e.target.value)}
        >
          {areas.map((a) => <option key={a} value={a}>{a === "all" ? "Todas as áreas" : a}</option>)}
        </select>
        <span className="text-xs" style={{ color: "var(--text-muted)" }}>
          {filtered.length} grupo{filtered.length !== 1 ? "s" : ""}
        </span>
      </div>

      {loading && (
        <div className="py-16 text-center" style={{ color: "var(--text-muted)" }}>Carregando…</div>
      )}
      {!loading && filtered.length === 0 && (
        <div className="py-16 text-center" style={{ color: "var(--text-muted)" }}>
          Nenhum grupo encontrado. Execute <code>make collect-dgp ingest-dgp</code>.
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filtered.map((g, i) => <GroupCard key={g.id || g.dgp_id || i} group={g} />)}
      </div>
    </div>
  );
}
