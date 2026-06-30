"use client";

import { useEffect, useState } from "react";
import { Users, Search, ExternalLink } from "lucide-react";
import { useRouter } from "next/navigation";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

const AREA_COLOR = {
  "Ciências Agrárias":   { dot: "#34D399", bg: "rgba(52,211,153,0.1)"  },
  "Ciências Biológicas": { dot: "#60A5FA", bg: "rgba(96,165,250,0.1)"  },
  "Ciências Exatas":     { dot: "#A78BFA", bg: "rgba(167,139,250,0.1)" },
  "Engenharias":         { dot: "#F59E0B", bg: "rgba(245,158,11,0.1)"  },
  "Ciências da Saúde":   { dot: "#FB7185", bg: "rgba(251,113,133,0.1)" },
};

function GroupCard({ group, onClick }) {
  const ac = AREA_COLOR[group.main_area] ?? { dot: "var(--text-dim)", bg: "var(--surface-2)" };

  return (
    <button
      onClick={onClick}
      className="w-full text-left rounded-xl p-5 flex flex-col gap-3 transition-all"
      style={{ background: "var(--surface)", border: "1px solid var(--border)", cursor: "pointer" }}
      onMouseEnter={(e) => { e.currentTarget.style.borderColor = ac.dot; e.currentTarget.style.background = "var(--surface-2)"; }}
      onMouseLeave={(e) => { e.currentTarget.style.borderColor = "var(--border)"; e.currentTarget.style.background = "var(--surface)"; }}
    >
      <div className="flex items-start justify-between gap-2">
        <h3 className="text-sm font-semibold text-white leading-snug">{group.name}</h3>
        <ExternalLink size={12} className="shrink-0 mt-0.5" style={{ color: "var(--text-dim)" }} />
      </div>

      {group.main_area && (
        <div className="flex items-center gap-1.5">
          <span className="w-2 h-2 rounded-full shrink-0" style={{ background: ac.dot }} />
          <span className="text-xs px-2 py-0.5 rounded-full" style={{ background: ac.bg, color: ac.dot }}>
            {group.main_area}
          </span>
        </div>
      )}

      {group.department && (
        <p className="text-xs" style={{ color: "var(--text-dim)" }}>{group.department}</p>
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
            <span className="text-[10px]" style={{ color: "var(--text-dim)" }}>
              +{group.research_lines.length - 3} linhas
            </span>
          )}
        </div>
      )}

      <p className="text-[10px] mt-auto" style={{ color: "var(--text-dim)" }}>
        Clique para consultar publicações no Oráculo →
      </p>
    </button>
  );
}

export default function GroupsPage() {
  const [groups, setGroups] = useState([]);
  const [loading, setLoad]  = useState(true);
  const [q, setQ]           = useState("");
  const [area, setArea]     = useState("all");
  const router = useRouter();

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

  function handleGroupClick(group) {
    const query = encodeURIComponent(`Publicações e pesquisadores do grupo ${group.name}`);
    router.push(`/oraculo?q=${query}`);
  }

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div>
        <h1 className="text-xl font-bold text-white flex items-center gap-2">
          <Users size={20} style={{ color: "var(--purple)" }} />
          Grupos de Pesquisa
        </h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Grupos UFV registrados no DGP/CNPq — clique para consultar publicações no Oráculo
        </p>
      </div>

      {/* Filtros por área */}
      {groups.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {areas.filter(a => a !== "all").map(a => {
            const ac = AREA_COLOR[a] ?? { dot: "var(--text-dim)", bg: "var(--surface-2)" };
            const count = groups.filter(g => g.main_area === a).length;
            return (
              <button key={a}
                onClick={() => setArea(area === a ? "all" : a)}
                className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-full transition-all"
                style={{
                  background: area === a ? ac.bg : "var(--surface)",
                  border: `1px solid ${area === a ? ac.dot : "var(--border)"}`,
                  color: area === a ? ac.dot : "var(--text-muted)"
                }}>
                <span className="w-1.5 h-1.5 rounded-full" style={{ background: ac.dot }} />
                {a} ({count})
              </button>
            );
          })}
        </div>
      )}

      <div className="flex gap-3 items-center flex-wrap">
        <div className="relative flex-1 min-w-48">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2" style={{ color: "var(--text-dim)" }} />
          <input
            className="w-full pl-8 pr-4 py-2 text-sm rounded-lg"
            style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
            placeholder="Buscar por nome ou líder…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
        <span className="text-xs" style={{ color: "var(--text-dim)" }}>
          {filtered.length} grupo{filtered.length !== 1 ? "s" : ""}
        </span>
      </div>

      {loading && (
        <div className="py-16 text-center" style={{ color: "var(--text-muted)" }}>Carregando…</div>
      )}
      {!loading && filtered.length === 0 && (
        <div className="py-16 text-center" style={{ color: "var(--text-muted)" }}>
          Nenhum grupo encontrado.
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filtered.map((g, i) => (
          <GroupCard key={g.id ?? g.dgp_id ?? i} group={g} onClick={() => handleGroupClick(g)} />
        ))}
      </div>
    </div>
  );
}
