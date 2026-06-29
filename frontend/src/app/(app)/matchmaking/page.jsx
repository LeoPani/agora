"use client";

import { useState, useEffect, useCallback } from "react";
import { GitMerge, RefreshCw, Building2, FlaskConical, ChevronDown, ChevronRight, CheckCircle2, Mail, X } from "lucide-react";
import { RadarLoader } from "@/components/loaders/RadarLoader";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

const SCORE_COLORS = {
  high:   { bg: "rgba(34,197,94,0.12)",  border: "rgba(34,197,94,0.3)",  text: "#4ade80"  },
  medium: { bg: "rgba(212,160,23,0.12)", border: "rgba(212,160,23,0.3)", text: "#D4A017"  },
  low:    { bg: "rgba(156,163,175,0.08)", border: "rgba(156,163,175,0.2)", text: "#9ca3af" },
};

function scoreLevel(score) {
  if (score >= 0.7) return "high";
  if (score >= 0.4) return "medium";
  return "low";
}

const STATUS_LABELS = {
  pending:     { label: "Pendente",     color: "var(--text-dim)" },
  contacted:   { label: "Contactado",   color: "var(--gold)"     },
  in_progress: { label: "Em andamento", color: "#60a5fa"         },
  closed:      { label: "Encerrado",    color: "#4ade80"         },
};

function ScoreBadge({ score }) {
  const level = scoreLevel(score);
  const c = SCORE_COLORS[level];
  return (
    <span
      className="text-xs font-bold px-2 py-0.5 rounded"
      style={{ background: c.bg, border: `1px solid ${c.border}`, color: c.text }}
    >
      {Math.round(score * 100)}%
    </span>
  );
}

function MatchCard({ match, onStatusChange }) {
  const [expanded, setExpanded]   = useState(false);
  const [updating, setUpdating]   = useState(false);
  const statusInfo = STATUS_LABELS[match.status] ?? STATUS_LABELS.pending;

  async function setStatus(status) {
    setUpdating(true);
    await fetch(`${API}/api/v1/matchmaking/${match.id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status }),
    }).catch(() => {});
    onStatusChange(match.id, status);
    setUpdating(false);
  }

  return (
    <div
      className="rounded-xl transition-all"
      style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
    >
      {/* Header */}
      <button
        className="w-full flex items-start gap-3 p-4 text-left"
        onClick={() => setExpanded((e) => !e)}
      >
        <div className="mt-0.5">
          {expanded ? <ChevronDown size={15} style={{ color: "var(--text-dim)" }} />
                    : <ChevronRight size={15} style={{ color: "var(--text-dim)" }} />}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-semibold text-white truncate">{match.partner_name}</span>
            <ScoreBadge score={match.score} />
            <span className="text-xs" style={{ color: statusInfo.color }}>{statusInfo.label}</span>
          </div>
          <p className="text-xs mt-0.5" style={{ color: "var(--text-dim)" }}>
            {match.sector} · {match.location}
          </p>
        </div>
      </button>

      {/* Expanded */}
      {expanded && (
        <div className="px-4 pb-4 space-y-3 border-t" style={{ borderColor: "var(--border)" }}>
          {match.reasons?.length > 0 && (
            <div className="mt-3">
              <p className="text-xs font-medium mb-1.5" style={{ color: "var(--text-muted)" }}>Motivos do match</p>
              <ul className="space-y-1">
                {match.reasons.map((r, i) => (
                  <li key={i} className="text-xs flex items-start gap-2" style={{ color: "var(--text-dim)" }}>
                    <CheckCircle2 size={11} className="mt-0.5 shrink-0" style={{ color: "var(--gold)" }} />
                    {r}
                  </li>
                ))}
              </ul>
            </div>
          )}
          <div className="flex gap-2 pt-1">
            {match.status !== "contacted" && (
              <button
                onClick={() => setStatus("contacted")}
                disabled={updating}
                className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg transition-all"
                style={{ background: "var(--purple)", color: "#fff", border: "none" }}
              >
                <Mail size={11} /> Marcar contactado
              </button>
            )}
            {match.status !== "in_progress" && (
              <button
                onClick={() => setStatus("in_progress")}
                disabled={updating}
                className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg transition-all"
                style={{ background: "rgba(96,165,250,0.15)", color: "#60a5fa", border: "1px solid rgba(96,165,250,0.3)" }}
              >
                Em andamento
              </button>
            )}
            {match.status !== "closed" && (
              <button
                onClick={() => setStatus("closed")}
                disabled={updating}
                className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg transition-all"
                style={{ background: "rgba(156,163,175,0.1)", color: "var(--text-dim)", border: "1px solid var(--border)" }}
              >
                <X size={11} /> Encerrar
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function DeptSection({ dept, matches }) {
  const [open, setOpen] = useState(true);
  const deptMatches = matches.filter((m) => m.department === dept.code);
  const topScore = deptMatches.length > 0 ? Math.max(...deptMatches.map((m) => m.score)) : 0;

  return (
    <div className="rounded-xl overflow-hidden" style={{ border: "1px solid var(--border)" }}>
      <button
        className="w-full flex items-center gap-3 px-5 py-3 text-left"
        style={{ background: "var(--surface-2)" }}
        onClick={() => setOpen((o) => !o)}
      >
        {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-xs font-bold tracking-widest" style={{ color: "var(--gold)" }}>
              {dept.code}
            </span>
            <span className="text-sm font-semibold text-white">
              {dept.groups.map((g) => g.name).join(" · ")}
            </span>
          </div>
          <p className="text-xs mt-0.5" style={{ color: "var(--text-dim)" }}>
            {dept.groups[0]?.main_area} · {deptMatches.length} parceiro{deptMatches.length !== 1 ? "s" : ""}
            {topScore > 0 && <span style={{ color: "var(--gold)" }}> · Melhor: {Math.round(topScore * 100)}%</span>}
          </p>
        </div>
      </button>

      {open && (
        <div className="p-4 space-y-2" style={{ background: "var(--surface)" }}>
          {/* Research lines */}
          <div className="flex flex-wrap gap-1.5 mb-3">
            {dept.groups.flatMap((g) => g.research_lines ?? []).slice(0, 6).map((line, i) => (
              <span
                key={i}
                className="text-xs px-2 py-0.5 rounded"
                style={{ background: "var(--purple-soft)", color: "var(--text-muted)", border: "1px solid var(--border)" }}
              >
                {line}
              </span>
            ))}
          </div>

          {deptMatches.length === 0 ? (
            <p className="text-xs py-2" style={{ color: "var(--text-dim)" }}>Nenhum match acima do limiar.</p>
          ) : (
            <div className="space-y-2">
              {deptMatches.map((m) => (
                <MatchCard key={m.id} match={m} onStatusChange={() => {}} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function MatchmakingPage() {
  const [depts, setDepts]     = useState([]);
  const [matches, setMatches] = useState([]);
  const [loading, setLoading] = useState(true);
  const [computing, setComputing] = useState(false);
  const [filter, setFilter]   = useState("all"); // all | pending | contacted | in_progress

  const load = useCallback(async () => {
    setLoading(true);
    const [d, m] = await Promise.all([
      fetch(`${API}/api/v1/departments`).then((r) => r.json()).catch(() => []),
      fetch(`${API}/api/v1/matchmaking`).then((r) => r.json()).catch(() => []),
    ]);
    setDepts(Array.isArray(d) ? d : []);
    setMatches(Array.isArray(m) ? m : []);
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  async function handleCompute() {
    setComputing(true);
    await fetch(`${API}/api/v1/matchmaking/compute`, { method: "POST" }).catch(() => {});
    await load();
    setComputing(false);
  }

  function updateMatchStatus(id, status) {
    setMatches((prev) => prev.map((m) => m.id === id ? { ...m, status } : m));
  }

  const filtered = filter === "all" ? matches : matches.filter((m) => m.status === filter);

  const stats = {
    total:       matches.length,
    pending:     matches.filter((m) => m.status === "pending").length,
    contacted:   matches.filter((m) => m.status === "contacted").length,
    in_progress: matches.filter((m) => m.status === "in_progress").length,
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <GitMerge size={22} style={{ color: "var(--purple)" }} />
          <div>
            <h1 className="text-lg font-semibold text-white">Matchmaking</h1>
            <p className="text-xs mt-0.5" style={{ color: "var(--text-dim)" }}>
              Grupos de pesquisa UFV × Parceiros potenciais
            </p>
          </div>
        </div>
        <button
          onClick={handleCompute}
          disabled={computing}
          className="flex items-center gap-2 text-sm px-4 py-2 rounded-lg transition-all"
          style={{ background: "var(--purple)", color: "#fff", border: "none", opacity: computing ? 0.7 : 1 }}
        >
          <RefreshCw size={14} className={computing ? "animate-spin" : ""} />
          {computing ? "Computando…" : "Recomputar matches"}
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-4 gap-3">
        {[
          { label: "Total",       value: stats.total,       color: "var(--text-muted)" },
          { label: "Pendentes",   value: stats.pending,     color: "var(--text-dim)"   },
          { label: "Contactados", value: stats.contacted,   color: "var(--gold)"       },
          { label: "Em andamento",value: stats.in_progress, color: "#60a5fa"           },
        ].map((s) => (
          <div
            key={s.label}
            className="rounded-xl p-4"
            style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
          >
            <p className="text-2xl font-bold" style={{ color: s.color }}>{s.value}</p>
            <p className="text-xs mt-0.5" style={{ color: "var(--text-dim)" }}>{s.label}</p>
          </div>
        ))}
      </div>

      {/* Filter */}
      <div className="flex gap-2">
        {["all", "pending", "contacted", "in_progress"].map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className="text-xs px-3 py-1.5 rounded-lg transition-all"
            style={{
              background: filter === f ? "var(--purple)" : "var(--surface)",
              color:      filter === f ? "#fff" : "var(--text-dim)",
              border:     `1px solid ${filter === f ? "transparent" : "var(--border)"}`,
            }}
          >
            {{ all: "Todos", pending: "Pendentes", contacted: "Contactados", in_progress: "Em andamento" }[f]}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <RadarLoader label="Carregando matches…" />
        </div>
      ) : (
        <div className="space-y-4">
          {depts.map((dept) => {
            const deptFiltered = { ...dept };
            const deptMatches = filtered.filter((m) => m.department === dept.code);
            if (deptMatches.length === 0 && filter !== "all") return null;
            return (
              <DeptSection
                key={dept.code}
                dept={dept}
                matches={deptMatches.length > 0 ? deptMatches : filtered.filter((m) => m.department === dept.code)}
              />
            );
          })}
        </div>
      )}
    </div>
  );
}
