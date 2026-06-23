"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Search, ExternalLink, Building2, User, Link2, BookOpen } from "lucide-react";

function LinkedInIcon({ size = 12 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor">
      <path d="M16 8a6 6 0 016 6v7h-4v-7a2 2 0 00-2-2 2 2 0 00-2 2v7h-4v-7a6 6 0 016-6z"/>
      <rect x="2" y="9" width="4" height="12"/>
      <circle cx="4" cy="4" r="2"/>
    </svg>
  );
}

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

const SOURCE_META = {
  cnpj:    { label: "Receita Federal", color: "#34D399", icon: Building2 },
  lattes:  { label: "Lattes/CNPq",     color: "#60A5FA", icon: BookOpen  },
  manual:  { label: "Manual",          color: "#FBBF24", icon: User      },
};

const TYPE_LABELS = {
  empresa:       "Empresa",
  pesquisador:   "Pesquisador",
  startup:       "Startup",
  ies:           "IES",
};

function ScoreRing({ value }) {
  const pct = Math.round((value || 0) * 100);
  const color = pct >= 70 ? "#34D399" : pct >= 40 ? "#FBBF24" : "#A78BFA";
  return (
    <div className="flex flex-col items-center gap-0.5">
      <div className="relative w-10 h-10">
        <svg viewBox="0 0 36 36" className="w-10 h-10 -rotate-90">
          <circle cx="18" cy="18" r="15.9" fill="none" stroke="var(--surface-2)" strokeWidth="3" />
          <circle cx="18" cy="18" r="15.9" fill="none"
            stroke={color} strokeWidth="3"
            strokeDasharray={`${pct} 100`}
            strokeLinecap="round" />
        </svg>
        <span className="absolute inset-0 flex items-center justify-center text-[10px] font-bold"
          style={{ color }}>{pct}</span>
      </div>
      <span className="text-[9px]" style={{ color: "var(--text-muted)" }}>score</span>
    </div>
  );
}

function PartnerCard({ partner }) {
  const src    = SOURCE_META[partner.source] || SOURCE_META.manual;
  const SrcIcon = src.icon;
  const type   = TYPE_LABELS[partner.partner_type] || partner.partner_type || "Parceiro";

  return (
    <div className="rounded-xl p-5 flex gap-4"
      style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>

      {/* Score ring */}
      <ScoreRing value={partner.interest_score} />

      {/* Info */}
      <div className="flex-1 min-w-0 space-y-2">
        <div className="flex items-start justify-between gap-2">
          <h3 className="text-sm font-semibold text-white leading-snug truncate">{partner.name}</h3>
          <Badge variant="muted">{type}</Badge>
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          {partner.sector && (
            <span className="text-xs px-2 py-0.5 rounded-full"
              style={{ background: "var(--surface-2)", color: "var(--text-muted)", border: "1px solid var(--border)" }}>
              {partner.sector}
            </span>
          )}
          {partner.location && (
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>{partner.location}</span>
          )}
        </div>

        <div className="flex items-center gap-3 flex-wrap">
          {/* Fonte */}
          <span className="flex items-center gap-1 text-xs" style={{ color: src.color }}>
            <SrcIcon size={11} />
            {src.label}
          </span>

          {partner.n_citations_to_ufv > 0 && (
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>
              {partner.n_citations_to_ufv} citações UFV
            </span>
          )}

          {partner.cnpj && (
            <span className="text-xs font-mono" style={{ color: "var(--text-muted)" }}>
              CNPJ: {partner.cnpj.replace(/(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})/, "$1.$2.$3/$4-$5")}
            </span>
          )}
        </div>

        {/* Links */}
        <div className="flex gap-3 pt-1">
          {partner.lattes_id && (
            <a href={`http://lattes.cnpq.br/${partner.lattes_id}`} target="_blank" rel="noreferrer"
              className="flex items-center gap-1 text-xs" style={{ color: "#60A5FA" }}>
              <BookOpen size={11} /> Lattes
            </a>
          )}
          {partner.linkedin_url && (
            <a href={partner.linkedin_url} target="_blank" rel="noreferrer"
              className="flex items-center gap-1 text-xs" style={{ color: "#0A66C2" }}>
              <LinkedInIcon size={11} /> LinkedIn
            </a>
          )}
          {partner.contact_email && (
            <a href={`mailto:${partner.contact_email}`}
              className="flex items-center gap-1 text-xs" style={{ color: "var(--accent-hover)" }}>
              <ExternalLink size={11} /> {partner.contact_email}
            </a>
          )}
        </div>
      </div>
    </div>
  );
}

function LinkedInLeadsPanel({ leads }) {
  if (!leads) return null;
  const [open, setOpen] = useState(false);
  return (
    <div className="rounded-xl overflow-hidden" style={{ border: "1px solid var(--border)" }}>
      <button
        className="w-full flex items-center justify-between px-5 py-3"
        style={{ background: "var(--surface)" }}
        onClick={() => setOpen((o) => !o)}
      >
        <div className="flex items-center gap-2">
          <LinkedInIcon size={14} />
          <span className="text-sm font-semibold text-white">Prospecção LinkedIn</span>
          <Badge variant="gold">{leads.total_queries} queries</Badge>
        </div>
        <span className="text-xs" style={{ color: "var(--text-muted)" }}>{open ? "▲" : "▼"}</span>
      </button>

      {open && (
        <div className="p-5 space-y-4" style={{ background: "var(--surface-2)" }}>
          <p className="text-xs" style={{ color: "var(--text-muted)" }}>{leads.instructions}</p>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {(leads.company_searches || []).slice(0, 12).map((s, i) => (
              <a key={i} href={s.linkedin_url} target="_blank" rel="noreferrer"
                className="flex flex-col gap-1 p-3 rounded-lg transition-colors"
                style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
                onMouseEnter={(e) => e.currentTarget.style.borderColor = "#0A66C2"}
                onMouseLeave={(e) => e.currentTarget.style.borderColor = "var(--border)"}
              >
                <div className="flex items-center gap-2">
                  <LinkedInIcon size={12} />
                  <span className="text-xs font-medium text-white truncate">{s.keyword}</span>
                </div>
                <div className="flex gap-1 flex-wrap">
                  {s.ufv_areas.slice(0, 2).map((a) => (
                    <span key={a} className="text-[10px] px-1.5 py-0.5 rounded"
                      style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}>{a}</span>
                  ))}
                </div>
              </a>
            ))}
          </div>
          <a href={`${API}/api/v1/linkedin-leads`} target="_blank" rel="noreferrer"
            className="text-xs" style={{ color: "var(--accent-hover)" }}>
            Ver todas as {leads.total_queries} queries em JSON →
          </a>
        </div>
      )}
    </div>
  );
}

export default function PartnersPage() {
  const [partners, setPartners] = useState([]);
  const [leads, setLeads]       = useState(null);
  const [loading, setLoad]      = useState(true);
  const [q, setQ]               = useState("");
  const [src, setSrc]           = useState("all");
  const [type, setType]         = useState("all");

  useEffect(() => {
    Promise.all([
      fetch(`${API}/api/v1/partners?limit=200`).then((r) => r.json()).catch(() => []),
      fetch(`${API}/api/v1/linkedin-leads`).then((r) => r.json()).catch(() => null),
    ]).then(([p, l]) => {
      setPartners(Array.isArray(p) ? p : []);
      setLeads(l);
      setLoad(false);
    });
  }, []);

  const sources = ["all", ...new Set(partners.map((p) => p.source).filter(Boolean))];
  const types   = ["all", ...new Set(partners.map((p) => p.partner_type).filter(Boolean))];

  const filtered = partners.filter((p) => {
    const matchQ = !q || (p.name || "").toLowerCase().includes(q.toLowerCase())
                       || (p.sector || "").toLowerCase().includes(q.toLowerCase());
    const matchS = src === "all" || p.source === src;
    const matchT = type === "all" || p.partner_type === type;
    return matchQ && matchS && matchT;
  });

  const kpis = [
    { label: "Parceiros",    value: partners.length },
    { label: "Empresas",     value: partners.filter((p) => p.partner_type === "empresa").length },
    { label: "Pesquisadores",value: partners.filter((p) => p.partner_type === "pesquisador").length },
    { label: "Score > 0.6",  value: partners.filter((p) => p.interest_score >= 0.6).length },
  ];

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Interessados</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Empresas e pesquisadores com interesse potencial nas pesquisas da UFV —
          via CNPJ/Receita Federal, Lattes/CNPq e LinkedIn
        </p>
      </div>

      {/* KPIs */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {kpis.map(({ label, value }) => (
          <div key={label} className="rounded-xl p-5" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
            <p className="text-xs font-semibold uppercase tracking-widest mb-2" style={{ color: "var(--text-muted)" }}>{label}</p>
            <p className="text-3xl font-bold text-white">{value}</p>
          </div>
        ))}
      </div>

      {/* LinkedIn leads */}
      <LinkedInLeadsPanel leads={leads} />

      {/* Filtros */}
      <div className="flex gap-3 items-center flex-wrap">
        <div className="relative flex-1 min-w-48">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2" style={{ color: "var(--text-muted)" }} />
          <input
            className="w-full pl-9 pr-4 py-2 text-sm rounded-lg"
            style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
            placeholder="Buscar por nome ou setor…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
        <select className="px-3 py-2 text-sm rounded-lg"
          style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
          value={src} onChange={(e) => setSrc(e.target.value)}>
          {sources.map((s) => <option key={s} value={s}>{s === "all" ? "Todas as fontes" : SOURCE_META[s]?.label || s}</option>)}
        </select>
        <select className="px-3 py-2 text-sm rounded-lg"
          style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
          value={type} onChange={(e) => setType(e.target.value)}>
          {types.map((t) => <option key={t} value={t}>{t === "all" ? "Todos os tipos" : TYPE_LABELS[t] || t}</option>)}
        </select>
        <span className="text-xs" style={{ color: "var(--text-muted)" }}>{filtered.length} parceiros</span>
      </div>

      {loading && (
        <div className="py-16 text-center" style={{ color: "var(--text-muted)" }}>Carregando…</div>
      )}
      {!loading && filtered.length === 0 && (
        <div className="py-16 text-center space-y-3">
          <p style={{ color: "var(--text-muted)" }}>Nenhum parceiro encontrado.</p>
          <div className="flex gap-2 justify-center flex-wrap">
            <code className="text-xs px-3 py-1.5 rounded"
              style={{ background: "var(--surface-2)", color: "var(--accent-hover)" }}>
              make collect-partners ingest-partners
            </code>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {filtered.map((p, i) => <PartnerCard key={p.id || i} partner={p} />)}
      </div>
    </div>
  );
}
