"use client";

import { useState, useEffect } from "react";
import { Bot, Copy, Check, Trash2, Zap, PlusCircle, ChevronDown, ChevronRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { RadarLoader } from "@/components/loaders/RadarLoader";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

const STATUS_VARIANTS = { draft: "gold", approved: "success", discarded: "muted" };
const STATUS_LABELS   = { draft: "Rascunho", approved: "Aprovado", discarded: "Descartado" };

function DraftCard({ draft, onStatusChange }) {
  const [copied,   setCopied]   = useState(false);
  const [expanded, setExpanded] = useState(false);

  function copyEmail() {
    const text = `Assunto: ${draft.subject}\n\n${draft.body}`;
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  const ctx = draft.context_used || {};
  const steps = ctx.context_log || [];

  return (
    <div
      className="rounded-xl overflow-hidden"
      style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
    >
      {/* Header */}
      <div className="flex items-start gap-3 p-5" style={{ borderBottom: "1px solid var(--border)" }}>
        <div
          className="w-8 h-8 rounded-full flex items-center justify-center shrink-0 mt-0.5"
          style={{ background: "var(--gold-soft)", border: "1px solid rgba(212,160,23,0.3)" }}
        >
          <Bot size={15} style={{ color: "var(--gold)" }} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <p className="text-sm font-semibold text-white truncate">{draft.subject}</p>
            <Badge variant={STATUS_VARIANTS[draft.status] ?? "muted"}>
              {STATUS_LABELS[draft.status] ?? draft.status}
            </Badge>
          </div>
          {draft.signal_title && (
            <p style={{ fontSize: 11, color: "var(--text-dim)", marginTop: 2 }}>
              Sinal: {draft.signal_title}
            </p>
          )}
          <p style={{ fontSize: 11, color: "var(--text-dim)" }}>
            {new Date(draft.created_at).toLocaleString("pt-BR")}
            {draft.cost_usd > 0 && ` · $${draft.cost_usd.toFixed(4)}`}
          </p>
        </div>
      </div>

      {/* Email body */}
      <div className="px-5 py-4">
        <pre
          className="text-sm leading-relaxed whitespace-pre-wrap"
          style={{ color: "var(--text-muted)", fontFamily: "inherit" }}
        >
          {draft.body}
        </pre>
      </div>

      {/* Context log (collapsible) */}
      {steps.length > 0 && (
        <div style={{ borderTop: "1px solid var(--border)" }}>
          <button
            onClick={() => setExpanded(!expanded)}
            className="w-full flex items-center gap-2 px-5 py-3 text-xs transition-all"
            style={{ color: "var(--text-dim)" }}
          >
            {expanded ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
            {steps.length} passo{steps.length > 1 ? "s" : ""} do agente
          </button>
          {expanded && (
            <div className="px-5 pb-4 space-y-2">
              {steps.map((s, i) => (
                <div
                  key={i}
                  className="text-xs rounded-lg px-3 py-2"
                  style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}
                >
                  <span style={{ color: "var(--accent-hover)", fontWeight: 600 }}>
                    [{s.step}] {s.tool}
                  </span>
                  {s.input && <p className="mt-1 opacity-70 truncate">Input: {s.input}</p>}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Actions */}
      {draft.status === "draft" && (
        <div
          className="flex items-center gap-2 px-5 py-3"
          style={{ borderTop: "1px solid var(--border)", background: "var(--surface-2)" }}
        >
          <button
            onClick={copyEmail}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
            style={{ background: "var(--purple)", color: "#fff" }}
          >
            {copied ? <Check size={13} /> : <Copy size={13} />}
            {copied ? "Copiado!" : "Copiar email"}
          </button>
          <button
            onClick={() => onStatusChange(draft.id, "approved")}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
            style={{ background: "rgba(52,211,153,0.15)", color: "#34D399", border: "1px solid rgba(52,211,153,0.3)" }}
          >
            Aprovar
          </button>
          <button
            onClick={() => onStatusChange(draft.id, "discarded")}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs transition-all ml-auto"
            style={{ color: "var(--text-dim)" }}
          >
            <Trash2 size={13} /> Descartar
          </button>
        </div>
      )}
    </div>
  );
}

function GenerateModal({ onGenerate, onClose }) {
  const [goal, setGoal] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleGenerate() {
    if (!goal.trim()) return;
    setLoading(true);
    await onGenerate(goal.trim());
    setLoading(false);
    onClose();
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      style={{ background: "rgba(0,0,0,0.7)" }}
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg rounded-2xl p-6"
        style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-white font-semibold mb-1">Gerar rascunho com agente</h2>
        <p style={{ fontSize: 12, color: "var(--text-muted)", marginBottom: 16 }}>
          Descreva o sinal de match. O agente pesquisará o contexto e gerará um email de aproximação.
        </p>
        <textarea
          autoFocus
          value={goal}
          onChange={(e) => setGoal(e.target.value)}
          placeholder="Ex: A Bayer citou 2 papers do Prof. João Silva (DFP/UFV) em patentes de defensivos biológicos. Nunca houve contato. Gere um email de aproximação."
          rows={5}
          className="w-full rounded-lg px-4 py-3 text-sm resize-none"
          style={{
            background: "var(--surface-2)",
            border:     "1px solid var(--border)",
            color:      "#fff",
          }}
        />
        <div className="flex gap-3 mt-4">
          <button
            onClick={handleGenerate}
            disabled={!goal.trim() || loading}
            className="flex-1 flex items-center justify-center gap-2 py-2.5 rounded-xl text-sm font-medium transition-all"
            style={{
              background: goal.trim() && !loading ? "var(--purple)" : "var(--surface-2)",
              color:      goal.trim() && !loading ? "#fff" : "var(--text-dim)",
            }}
          >
            {loading ? <RadarLoader size="sm" /> : <><Zap size={14} /> Gerar email</>}
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2.5 rounded-xl text-sm transition-all"
            style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}
          >
            Cancelar
          </button>
        </div>
        <p style={{ fontSize: 10, color: "var(--text-dim)", marginTop: 10 }}>
          Custo estimado por rascunho: $0.05–0.15 · Limite diário: $5.00
        </p>
      </div>
    </div>
  );
}

export default function AgentePage() {
  const [drafts,      setDrafts]      = useState([]);
  const [loading,     setLoading]     = useState(true);
  const [generating,  setGenerating]  = useState(false);
  const [showModal,   setShowModal]   = useState(false);
  const [genError,    setGenError]    = useState(null);

  useEffect(() => { loadDrafts(); }, []);

  async function loadDrafts() {
    setLoading(true);
    const data = await fetch(`${API}/api/v1/agent-drafts`)
      .then((r) => r.json())
      .catch(() => []);
    setDrafts(Array.isArray(data) ? data : []);
    setLoading(false);
  }

  async function handleStatusChange(id, status) {
    await fetch(`${API}/api/v1/agent-drafts/${id}`, {
      method:  "PATCH",
      headers: { "Content-Type": "application/json" },
      body:    JSON.stringify({ status }),
    });
    setDrafts((prev) => prev.map((d) => d.id === id ? { ...d, status } : d));
  }

  async function handleGenerate(goal) {
    setGenerating(true);
    setGenError(null);
    try {
      const resp = await fetch(`${API}/api/v1/agent-drafts/generate`, {
        method:  "POST",
        headers: { "Content-Type": "application/json" },
        body:    JSON.stringify({ goal }),
      });
      const data = await resp.json();
      if (!resp.ok) {
        setGenError(data.error ?? "Erro ao gerar rascunho");
      } else {
        await loadDrafts();
      }
    } catch (e) {
      setGenError(e.message);
    } finally {
      setGenerating(false);
    }
  }

  const pendingCount = drafts.filter((d) => d.status === "draft").length;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-4">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold text-white">Agente</h1>
            <span
              className="px-2 py-0.5 rounded text-xs font-bold"
              style={{ background: "var(--gold-soft)", color: "var(--gold)", border: "1px solid rgba(212,160,23,0.3)" }}
            >
              AI
            </span>
          </div>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Rascunhos de email gerados automaticamente pelo agente de TT
          </p>
        </div>
        <button
          onClick={() => setShowModal(true)}
          className="flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium transition-all"
          style={{ background: "var(--purple)", color: "#fff" }}
          onMouseEnter={(e) => { e.currentTarget.style.background = "var(--accent-hover)"; }}
          onMouseLeave={(e) => { e.currentTarget.style.background = "var(--purple)"; }}
        >
          <PlusCircle size={15} /> Gerar rascunho
        </button>
      </div>

      {/* Stats bar */}
      {drafts.length > 0 && (
        <div
          className="flex items-center gap-6 px-5 py-3 rounded-xl text-sm"
          style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
        >
          <span className="text-white font-medium">{drafts.length} rascunho{drafts.length > 1 ? "s" : ""}</span>
          <span style={{ color: "#FBBF24" }}>{pendingCount} pendente{pendingCount > 1 ? "s" : ""}</span>
          <span style={{ color: "#34D399" }}>
            {drafts.filter((d) => d.status === "approved").length} aprovado{drafts.filter((d) => d.status === "approved").length > 1 ? "s" : ""}
          </span>
          <span style={{ color: "var(--text-dim)" }} className="ml-auto text-xs">
            Custo total: ${drafts.reduce((s, d) => s + (d.cost_usd || 0), 0).toFixed(4)}
          </span>
        </div>
      )}

      {genError && (
        <div
          className="px-4 py-3 rounded-xl text-sm"
          style={{ background: "rgba(251,113,133,0.1)", border: "1px solid rgba(251,113,133,0.3)", color: "#FB7185" }}
        >
          {genError}
        </div>
      )}

      {/* Content */}
      {loading ? (
        <div className="flex justify-center py-16">
          <RadarLoader size="lg" label="Carregando rascunhos…" />
        </div>
      ) : generating ? (
        <div className="flex flex-col items-center py-16 gap-4">
          <RadarLoader size="lg" />
          <p style={{ color: "var(--text-muted)", fontSize: 14 }}>
            Agente pesquisando contexto e gerando email…
          </p>
          <p style={{ color: "var(--text-dim)", fontSize: 12 }}>Isso pode levar 20-60 segundos</p>
        </div>
      ) : drafts.length === 0 ? (
        <div
          className="flex flex-col items-center justify-center rounded-2xl py-20 gap-4"
          style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
        >
          <Bot size={40} style={{ color: "var(--text-dim)" }} />
          <p className="text-white font-medium">Nenhum rascunho gerado ainda</p>
          <p style={{ fontSize: 13, color: "var(--text-muted)" }}>
            Clique em "Gerar rascunho" para criar o primeiro email de aproximação
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {drafts.map((d) => (
            <DraftCard key={d.id} draft={d} onStatusChange={handleStatusChange} />
          ))}
        </div>
      )}

      {showModal && (
        <GenerateModal
          onGenerate={async (goal) => {
            setShowModal(false);
            setGenerating(true);
            await handleGenerate(goal);
          }}
          onClose={() => setShowModal(false)}
        />
      )}
    </div>
  );
}
