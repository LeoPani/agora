"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { BookOpen, Search } from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

export default function PublicationsPage() {
  const [pubs, setPubs]       = useState([]);
  const [loading, setLoad]    = useState(true);
  const [q, setQ]             = useState("");
  const [source, setSource]   = useState("all");
  const [modal, setModal]     = useState(null);

  useEffect(() => {
    fetch(`${API}/api/v1/publications?limit=200`)
      .then((r) => r.json())
      .catch(() => [])
      .then((d) => { setPubs(Array.isArray(d) ? d : d?.publications ?? []); setLoad(false); });
  }, []);

  const sources = ["all", ...new Set(pubs.map((p) => p.source || "OpenAlex"))];

  const filtered = pubs.filter((p) => {
    const matchQ = !q || (p.title || "").toLowerCase().includes(q.toLowerCase());
    const matchS = source === "all" || (p.source || "OpenAlex") === source;
    return matchQ && matchS;
  });

  const bySource = pubs.reduce((acc, p) => {
    const s = p.source || "OpenAlex";
    acc[s] = (acc[s] || 0) + 1;
    return acc;
  }, {});

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Publicações</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Produção científica UFV indexada
        </p>
      </div>

      {/* KPIs por fonte */}
      <div className="flex gap-4 flex-wrap">
        {Object.entries(bySource).map(([s, n]) => (
          <div key={s} className="rounded-xl px-5 py-3 flex gap-3 items-center"
            style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
            <BookOpen size={14} style={{ color: "var(--accent-hover)" }} />
            <span className="text-sm font-medium text-white">{s}</span>
            <span className="text-xl font-bold text-white">{n.toLocaleString("pt-BR")}</span>
          </div>
        ))}
        <div className="rounded-xl px-5 py-3 flex gap-3 items-center"
          style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
          <span className="text-sm" style={{ color: "var(--text-muted)" }}>Total</span>
          <span className="text-xl font-bold text-white">{pubs.length.toLocaleString("pt-BR")}</span>
        </div>
      </div>

      {/* Filtros */}
      <div className="flex gap-3 flex-wrap items-center">
        <div className="relative flex-1 min-w-48">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2" style={{ color: "var(--text-muted)" }} />
          <input
            className="w-full pl-9 pr-4 py-2 text-sm rounded-lg"
            style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
            placeholder="Buscar por título…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
        <select
          className="px-3 py-2 text-sm rounded-lg"
          style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
          value={source}
          onChange={(e) => setSource(e.target.value)}
        >
          {sources.map((s) => <option key={s} value={s}>{s === "all" ? "Todas as fontes" : s}</option>)}
        </select>
      </div>

      {/* Tabela */}
      <div className="rounded-xl overflow-hidden" style={{ border: "1px solid var(--border)" }}>
        <table className="w-full text-sm">
          <thead>
            <tr style={{ background: "var(--surface)" }}>
              {["Título", "Ano", "Tipo", "Citações", "Fonte"].map((h) => (
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
              <tr><td colSpan={5} className="text-center py-8" style={{ color: "var(--text-muted)" }}>Nenhuma publicação encontrada.</td></tr>
            )}
            {filtered.slice(0, 100).map((p, i) => (
              <tr
                key={p.id || p.openalex_id || i}
                className="cursor-pointer transition-colors"
                style={{ borderBottom: "1px solid var(--border)" }}
                onClick={() => setModal(p)}
                onMouseEnter={(e) => e.currentTarget.style.background = "var(--surface-2)"}
                onMouseLeave={(e) => e.currentTarget.style.background = "transparent"}
              >
                <td className="px-4 py-3 text-white max-w-md truncate">{p.title || "—"}</td>
                <td className="px-4 py-3 tabular-nums" style={{ color: "var(--text-muted)" }}>{p.publication_year || "—"}</td>
                <td className="px-4 py-3"><Badge variant="muted">{p.type || "article"}</Badge></td>
                <td className="px-4 py-3 tabular-nums" style={{ color: "var(--text-muted)" }}>{p.cited_by_count ?? 0}</td>
                <td className="px-4 py-3"><Badge variant="default">{p.source || "OpenAlex"}</Badge></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {filtered.length > 100 && (
        <p className="text-xs text-center" style={{ color: "var(--text-muted)" }}>
          Mostrando 100 de {filtered.length.toLocaleString("pt-BR")} resultados. Use a busca para refinar.
        </p>
      )}

      {/* Modal */}
      {modal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center p-4"
          style={{ background: "rgba(0,0,0,0.7)" }}
          onClick={() => setModal(null)}
        >
          <div
            className="rounded-xl p-6 max-w-xl w-full max-h-[80vh] overflow-y-auto"
            style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-base font-bold text-white mb-3">{modal.title}</h2>
            <div className="flex gap-2 flex-wrap mb-4">
              <Badge variant="muted">{modal.type || "article"}</Badge>
              <Badge variant="muted">{modal.publication_year}</Badge>
              <Badge variant={modal.cited_by_count > 50 ? "gold" : "default"}>
                {modal.cited_by_count ?? 0} citações
              </Badge>
            </div>
            {modal.abstract && (
              <p className="text-sm leading-relaxed mb-4" style={{ color: "var(--text-muted)" }}>
                {modal.abstract}
              </p>
            )}
            {modal.doi && (
              <a href={`https://doi.org/${modal.doi}`} target="_blank" rel="noreferrer"
                className="text-xs" style={{ color: "var(--accent-hover)" }}>
                DOI: {modal.doi} ↗
              </a>
            )}
            <button
              className="mt-4 text-xs px-4 py-2 rounded-lg block"
              style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}
              onClick={() => setModal(null)}
            >
              Fechar
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
