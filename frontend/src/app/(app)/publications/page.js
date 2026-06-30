"use client";

import { useEffect, useState, useMemo } from "react";
import { BookOpen, Search, ExternalLink, TrendingUp, Award, Filter, X } from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

const TYPE_LABELS = {
  article: "Artigo",
  "book-chapter": "Capítulo",
  review: "Revisão",
  preprint: "Preprint",
  dissertation: "Dissertação",
  dataset: "Dataset",
  editorial: "Editorial",
  other: "Outro",
};

const TYPE_COLOR = {
  article:    { bg: "rgba(96,165,250,0.12)",  text: "#60a5fa" },
  review:     { bg: "rgba(167,139,250,0.12)", text: "#a78bfa" },
  preprint:   { bg: "rgba(212,160,23,0.12)",  text: "#D4A017" },
  dissertation:{ bg:"rgba(52,211,153,0.12)",  text: "#34d399" },
};

function PubCard({ pub, onClick }) {
  const typeC = TYPE_COLOR[pub.type] ?? { bg: "rgba(156,163,175,0.1)", text: "#9ca3af" };
  const year  = pub.publication_year;
  const cited = pub.cited_by_count ?? 0;
  const title = (pub.title || "").replace(/<[^>]+>/g, "");
  const abstract = (pub.abstract || "").replace(/<[^>]+>/g, "").slice(0, 180);

  return (
    <button
      onClick={onClick}
      className="w-full text-left rounded-xl p-5 flex flex-col gap-3 transition-all"
      style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
      onMouseEnter={(e) => { e.currentTarget.style.borderColor = "var(--purple)"; e.currentTarget.style.background = "var(--surface-2)"; }}
      onMouseLeave={(e) => { e.currentTarget.style.borderColor = "var(--border)"; e.currentTarget.style.background = "var(--surface)"; }}
    >
      <div className="flex items-start justify-between gap-2">
        <span
          className="text-[10px] font-semibold px-2 py-0.5 rounded-full shrink-0"
          style={{ background: typeC.bg, color: typeC.text }}
        >
          {TYPE_LABELS[pub.type] ?? pub.type}
        </span>
        <span className="text-xs tabular-nums shrink-0" style={{ color: "var(--text-dim)" }}>{year}</span>
      </div>

      <h3 className="text-sm font-semibold text-white leading-snug line-clamp-2"
        style={{ display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical", overflow: "hidden" }}>
        {title}
      </h3>

      {abstract && (
        <p className="text-xs leading-relaxed"
          style={{ color: "var(--text-dim)", display: "-webkit-box", WebkitLineClamp: 3, WebkitBoxOrient: "vertical", overflow: "hidden" }}>
          {abstract}…
        </p>
      )}

      {cited > 0 && (
        <div className="flex items-center gap-1.5 mt-auto">
          <TrendingUp size={11} style={{ color: cited > 100 ? "var(--gold)" : "var(--text-dim)" }} />
          <span className="text-xs tabular-nums" style={{ color: cited > 100 ? "var(--gold)" : "var(--text-dim)" }}>
            {cited.toLocaleString("pt-BR")} citações
          </span>
        </div>
      )}
    </button>
  );
}

function StatCard({ label, value, sub, color }) {
  return (
    <div className="rounded-xl p-4" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
      <p className="text-2xl font-bold" style={{ color: color ?? "var(--text-muted)" }}>
        {typeof value === "number" ? value.toLocaleString("pt-BR") : value}
      </p>
      <p className="text-xs font-medium mt-0.5 text-white">{label}</p>
      {sub && <p className="text-[10px] mt-0.5" style={{ color: "var(--text-dim)" }}>{sub}</p>}
    </div>
  );
}

function Modal({ pub, onClose }) {
  if (!pub) return null;
  const title    = (pub.title || "").replace(/<[^>]+>/g, "");
  const abstract = (pub.abstract || "").replace(/<[^>]+>/g, "");
  const typeC    = TYPE_COLOR[pub.type] ?? { bg: "rgba(156,163,175,0.1)", text: "#9ca3af" };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4"
      style={{ background: "rgba(0,0,0,0.75)" }} onClick={onClose}>
      <div className="rounded-2xl p-6 max-w-2xl w-full max-h-[85vh] overflow-y-auto"
        style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
        onClick={(e) => e.stopPropagation()}>
        <div className="flex items-start justify-between gap-3 mb-4">
          <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full"
            style={{ background: typeC.bg, color: typeC.text }}>
            {TYPE_LABELS[pub.type] ?? pub.type}
          </span>
          <button onClick={onClose} className="shrink-0" style={{ color: "var(--text-dim)" }}>
            <X size={16} />
          </button>
        </div>

        <h2 className="text-base font-bold text-white leading-snug mb-3">{title}</h2>

        <div className="flex flex-wrap gap-2 mb-4">
          {pub.publication_year && (
            <span className="text-xs px-2 py-0.5 rounded" style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}>
              {pub.publication_year}
            </span>
          )}
          {(pub.cited_by_count ?? 0) > 0 && (
            <span className="text-xs px-2 py-0.5 rounded flex items-center gap-1"
              style={{ background: "rgba(212,160,23,0.1)", color: "var(--gold)" }}>
              <Award size={10} /> {pub.cited_by_count.toLocaleString("pt-BR")} citações
            </span>
          )}
        </div>

        {abstract && (
          <p className="text-sm leading-relaxed mb-4" style={{ color: "var(--text-muted)" }}>{abstract}</p>
        )}

        {pub.doi && (
          <a href={`https://doi.org/${pub.doi.replace("https://doi.org/","")}`}
            target="_blank" rel="noreferrer"
            className="inline-flex items-center gap-1.5 text-xs py-2 px-3 rounded-lg"
            style={{ background: "var(--purple-soft)", color: "var(--purple)", border: "1px solid rgba(139,92,246,0.3)" }}>
            <ExternalLink size={11} /> Ver publicação ↗
          </a>
        )}
      </div>
    </div>
  );
}

const YEARS = ["all", ...Array.from({ length: 15 }, (_, i) => String(2024 - i))];
const TYPES = ["all", "article", "review", "book-chapter", "preprint", "dissertation", "other"];

export default function PublicationsPage() {
  const [pubs, setPubs]     = useState([]);
  const [loading, setLoad]  = useState(true);
  const [q, setQ]           = useState("");
  const [year, setYear]     = useState("all");
  const [type, setType]     = useState("all");
  const [modal, setModal]   = useState(null);
  const [page, setPage]     = useState(0);
  const PER_PAGE = 30;

  useEffect(() => {
    fetch(`${API}/api/v1/publications?limit=500`)
      .then((r) => r.json())
      .catch(() => [])
      .then((d) => { setPubs(Array.isArray(d) ? d : []); setLoad(false); });
  }, []);

  // Stats
  const stats = useMemo(() => {
    const totalCitations = pubs.reduce((s, p) => s + (p.cited_by_count ?? 0), 0);
    const topCited = [...pubs].sort((a, b) => (b.cited_by_count ?? 0) - (a.cited_by_count ?? 0)).slice(0, 3);
    const byYear = pubs.reduce((a, p) => { const y = p.publication_year; if (y) a[y] = (a[y]||0)+1; return a; }, {});
    const peakYear = Object.entries(byYear).sort((a,b)=>b[1]-a[1])[0];
    return { totalCitations, topCited, peakYear };
  }, [pubs]);

  const filtered = useMemo(() => {
    return pubs.filter((p) => {
      const title = ((p.title || "") + " " + (p.abstract || "")).toLowerCase();
      if (q && !title.includes(q.toLowerCase())) return false;
      if (year !== "all" && String(p.publication_year) !== year) return false;
      if (type !== "all") {
        const t = p.type || "other";
        if (type === "other") return !["article","review","book-chapter","preprint","dissertation"].includes(t);
        if (t !== type) return false;
      }
      return true;
    });
  }, [pubs, q, year, type]);

  const hasFilters = q || year !== "all" || type !== "all";
  const pageSlice  = filtered.slice(page * PER_PAGE, (page + 1) * PER_PAGE);
  const totalPages = Math.ceil(filtered.length / PER_PAGE);

  function resetFilters() { setQ(""); setYear("all"); setType("all"); setPage(0); }

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-xl font-bold text-white flex items-center gap-2">
            <BookOpen size={20} style={{ color: "var(--purple)" }} />
            Portfólio de Publicações UFV
          </h1>
          <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>
            Produção científica indexada pela OpenAlex
          </p>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <StatCard label="Publicações" value={pubs.length} color="var(--purple)" />
        <StatCard label="Citações totais" value={stats.totalCitations} color="var(--gold)" />
        <StatCard label="Artigos" value={pubs.filter(p=>p.type==="article").length} color="#60a5fa" />
        <StatCard label="Ano mais produtivo" value={stats.peakYear?.[0] ?? "—"}
          sub={stats.peakYear ? `${stats.peakYear[1]} publicações` : ""} color="#34d399" />
      </div>

      {/* Top citados */}
      {stats.topCited.length > 0 && (
        <div className="rounded-xl p-4" style={{ background: "var(--surface)", border: "1px solid rgba(212,160,23,0.25)" }}>
          <p className="text-xs font-semibold mb-3 flex items-center gap-1.5" style={{ color: "var(--gold)" }}>
            <Award size={12} /> Mais citados
          </p>
          <div className="space-y-2">
            {stats.topCited.map((p, i) => (
              <button key={p.id} onClick={() => setModal(p)}
                className="w-full text-left flex items-start gap-3 text-xs"
                style={{ color: "var(--text-muted)" }}>
                <span className="font-bold tabular-nums shrink-0" style={{ color: "var(--gold)" }}>#{i+1}</span>
                <span className="flex-1 text-white line-clamp-1 truncate">{(p.title||"").replace(/<[^>]+>/g,"")}</span>
                <span className="shrink-0 tabular-nums">{(p.cited_by_count??0).toLocaleString("pt-BR")} cit.</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Filtros */}
      <div className="flex gap-2 flex-wrap items-center">
        <div className="relative flex-1 min-w-48">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2" style={{ color: "var(--text-dim)" }} />
          <input
            className="w-full pl-8 pr-4 py-2 text-sm rounded-lg"
            style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
            placeholder="Buscar por título ou palavra-chave…"
            value={q}
            onChange={(e) => { setQ(e.target.value); setPage(0); }}
          />
        </div>
        <select value={year} onChange={(e) => { setYear(e.target.value); setPage(0); }}
          className="px-3 py-2 text-sm rounded-lg"
          style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}>
          {YEARS.map((y) => <option key={y} value={y}>{y === "all" ? "Todos os anos" : y}</option>)}
        </select>
        <select value={type} onChange={(e) => { setType(e.target.value); setPage(0); }}
          className="px-3 py-2 text-sm rounded-lg"
          style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}>
          {TYPES.map((t) => <option key={t} value={t}>{t === "all" ? "Todos os tipos" : (TYPE_LABELS[t] ?? t)}</option>)}
        </select>
        {hasFilters && (
          <button onClick={resetFilters} className="flex items-center gap-1 text-xs px-3 py-2 rounded-lg"
            style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text-dim)" }}>
            <X size={12} /> Limpar
          </button>
        )}
        <span className="text-xs ml-auto" style={{ color: "var(--text-dim)" }}>
          {filtered.length.toLocaleString("pt-BR")} resultado{filtered.length !== 1 ? "s" : ""}
        </span>
      </div>

      {/* Grid de cards */}
      {loading ? (
        <div className="py-16 text-center" style={{ color: "var(--text-muted)" }}>Carregando portfólio…</div>
      ) : filtered.length === 0 ? (
        <div className="py-16 text-center" style={{ color: "var(--text-muted)" }}>Nenhuma publicação encontrada.</div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {pageSlice.map((p) => (
              <PubCard key={p.id ?? p.openalex_id} pub={p} onClick={() => setModal(p)} />
            ))}
          </div>

          {/* Paginação */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 pt-2">
              <button disabled={page === 0}
                onClick={() => setPage(p => p - 1)}
                className="px-3 py-1.5 text-xs rounded-lg disabled:opacity-40"
                style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text-muted)" }}>
                ← Anterior
              </button>
              <span className="text-xs" style={{ color: "var(--text-dim)" }}>
                {page + 1} / {totalPages}
              </span>
              <button disabled={page >= totalPages - 1}
                onClick={() => setPage(p => p + 1)}
                className="px-3 py-1.5 text-xs rounded-lg disabled:opacity-40"
                style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text-muted)" }}>
                Próxima →
              </button>
            </div>
          )}
        </>
      )}

      <Modal pub={modal} onClose={() => setModal(null)} />
    </div>
  );
}
