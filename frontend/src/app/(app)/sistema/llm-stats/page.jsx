"use client";

import { useEffect, useState } from "react";
import { Activity, DollarSign, Zap, Clock, AlertTriangle, RefreshCw } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { RadarLoader } from "@/components/loaders/RadarLoader";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

function KPI({ icon: Icon, label, value, sub, color }) {
  return (
    <div
      className="rounded-xl p-5 flex flex-col gap-3"
      style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
    >
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold uppercase tracking-widest" style={{ color: "var(--text-muted)" }}>
          {label}
        </span>
        <div className="p-2 rounded-lg" style={{ background: "var(--surface-2)" }}>
          <Icon size={16} style={{ color: color || "var(--accent-hover)" }} />
        </div>
      </div>
      <div>
        <p className="text-3xl font-bold text-white">{value}</p>
        {sub && <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>{sub}</p>}
      </div>
    </div>
  );
}

function PurposeBar({ items }) {
  if (!items || items.length === 0) {
    return <p style={{ fontSize: 13, color: "var(--text-dim)" }}>Sem dados</p>;
  }
  const maxCost = Math.max(...items.map((i) => i.cost_usd), 0.001);
  return (
    <div className="space-y-3">
      {items.map((item) => (
        <div key={item.purpose}>
          <div className="flex items-center justify-between mb-1">
            <span style={{ fontSize: 12, color: "var(--text-muted)" }}>{item.purpose}</span>
            <div className="flex items-center gap-3">
              <span style={{ fontSize: 11, color: "var(--text-dim)" }}>{item.calls} calls</span>
              <span style={{ fontSize: 12, fontWeight: 600, color: "#FBBF24" }}>
                ${item.cost_usd.toFixed(4)}
              </span>
            </div>
          </div>
          <div className="h-1.5 rounded-full overflow-hidden" style={{ background: "var(--surface-2)" }}>
            <div
              className="h-full rounded-full transition-all"
              style={{
                width:      `${(item.cost_usd / maxCost) * 100}%`,
                background: "linear-gradient(90deg, var(--purple), var(--accent-hover))",
              }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

const PROVIDER_COLORS = {
  groq:      "#6B21A8",
  gemini:    "#60A5FA",
  anthropic: "#34D399",
  ollama:    "#FBBF24",
};

export default function LLMStatsPage() {
  const [stats,   setStats]   = useState(null);
  const [loading, setLoading] = useState(true);

  async function load() {
    setLoading(true);
    const data = await fetch(`${API}/api/v1/llm-stats`)
      .then((r) => r.json())
      .catch(() => null);
    setStats(data);
    setLoading(false);
  }

  useEffect(() => { load(); }, []);

  const fmt = (n, dec = 4) => n != null ? `$${Number(n).toFixed(dec)}` : "—";
  const fmtMs = (n) => n != null ? `${Math.round(n)} ms` : "—";
  const fmtPct = (n) => n != null ? `${(Number(n) * 100).toFixed(1)}%` : "—";
  const fmtInt = (n) => n != null ? Number(n).toLocaleString("pt-BR") : "—";

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">LLM Stats</h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Custo, latência e qualidade das chamadas de IA
          </p>
        </div>
        <button
          onClick={load}
          disabled={loading}
          className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-all"
          style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text-muted)" }}
        >
          <RefreshCw size={14} className={loading ? "animate-spin" : ""} />
          Atualizar
        </button>
      </div>

      {loading && !stats ? (
        <div className="flex justify-center py-16">
          <RadarLoader size="lg" label="Carregando estatísticas…" />
        </div>
      ) : !stats ? (
        <p style={{ color: "var(--text-muted)" }}>Nenhum dado disponível. Faça chamadas LLM primeiro.</p>
      ) : (
        <>
          {/* KPIs */}
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <KPI icon={DollarSign}    label="Custo Hoje"      value={fmt(stats.today_cost_usd, 4)}  sub="USD"                 color="#FBBF24" />
            <KPI icon={DollarSign}    label="Custo Mês"       value={fmt(stats.month_cost_usd, 2)}  sub="USD"                 color="#F59E0B" />
            <KPI icon={Zap}           label="Chamadas Hoje"   value={fmtInt(stats.total_calls)}      sub="Total"              color="#A78BFA" />
            <KPI icon={Clock}         label="Latência Média"  value={fmtMs(stats.avg_latency_ms)}   sub="por chamada"         color="#60A5FA" />
            <KPI icon={AlertTriangle} label="Taxa de Erro"    value={fmtPct(stats.error_rate)}       sub="nas últimas 24h"    color="#FB7185" />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Custo por purpose */}
            <div
              className="rounded-xl p-5"
              style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
            >
              <div className="flex items-center gap-2 mb-4">
                <Activity size={16} style={{ color: "var(--accent-hover)" }} />
                <h2 className="text-sm font-semibold text-white">Custo por Finalidade (30 dias)</h2>
              </div>
              <PurposeBar items={stats.by_purpose} />
            </div>

            {/* Info de provedor */}
            <div
              className="rounded-xl p-5"
              style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
            >
              <div className="flex items-center gap-2 mb-4">
                <Zap size={16} style={{ color: "var(--gold)" }} />
                <h2 className="text-sm font-semibold text-white">Provedores Configurados</h2>
              </div>
              <div className="space-y-3">
                {[
                  { name: "Groq",      hint: "GROQ_API_KEY",      speed: "Rápido",  cost: "~$0.00/1K" },
                  { name: "Gemini",    hint: "GEMINI_API_KEY",    speed: "Médio",   cost: "~$0.001/1K" },
                  { name: "Anthropic", hint: "ANTHROPIC_API_KEY", speed: "Lento",   cost: "~$0.01/1K"  },
                  { name: "Ollama",    hint: "local",             speed: "Variável", cost: "Grátis"    },
                ].map((p) => (
                  <div key={p.name} className="flex items-center gap-3">
                    <div
                      className="w-2 h-2 rounded-full"
                      style={{ background: PROVIDER_COLORS[p.name.toLowerCase()] ?? "#fff" }}
                    />
                    <span className="text-sm text-white font-medium w-20">{p.name}</span>
                    <code style={{ fontSize: 10, color: "var(--text-dim)" }}>{p.hint}</code>
                    <span className="ml-auto text-xs" style={{ color: "var(--text-muted)" }}>{p.cost}</span>
                  </div>
                ))}
              </div>
              <div
                className="mt-4 p-3 rounded-lg text-xs"
                style={{ background: "var(--surface-2)", color: "var(--text-dim)" }}
              >
                Limite diário padrão: $5.00 (LLM_COST_LIMIT_DAILY_USD)
              </div>
            </div>
          </div>

          {/* Tabela de chamadas recentes */}
          <div
            className="rounded-xl"
            style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
          >
            <div className="px-5 py-4" style={{ borderBottom: "1px solid var(--border)" }}>
              <h2 className="text-sm font-semibold text-white">Chamadas Recentes</h2>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr style={{ borderBottom: "1px solid var(--border)" }}>
                    {["Purpose", "Provider / Model", "Tokens", "Custo", "Latência", "Status", "Hora"].map((h) => (
                      <th
                        key={h}
                        className="text-left px-4 py-3 text-xs font-semibold uppercase tracking-wider"
                        style={{ color: "var(--text-muted)" }}
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {(stats.recent_calls ?? []).slice(0, 20).map((c) => (
                    <tr
                      key={c.id}
                      style={{ borderBottom: "1px solid var(--border)" }}
                      onMouseEnter={(e) => { e.currentTarget.style.background = "var(--surface-2)"; }}
                      onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
                    >
                      <td className="px-4 py-3 text-white">{c.purpose}</td>
                      <td className="px-4 py-3">
                        <div style={{ color: PROVIDER_COLORS[c.provider] ?? "var(--text-muted)", fontSize: 12 }}>
                          {c.provider}
                        </div>
                        <div style={{ fontSize: 10, color: "var(--text-dim)", marginTop: 1 }}>
                          {c.model}
                        </div>
                      </td>
                      <td className="px-4 py-3 tabular-nums" style={{ color: "var(--text-muted)" }}>
                        {(c.total_tokens ?? 0).toLocaleString("pt-BR")}
                      </td>
                      <td className="px-4 py-3 tabular-nums" style={{ color: "#FBBF24" }}>
                        ${(c.cost_usd ?? 0).toFixed(5)}
                      </td>
                      <td className="px-4 py-3 tabular-nums" style={{ color: "var(--text-muted)" }}>
                        {c.latency_ms} ms
                      </td>
                      <td className="px-4 py-3">
                        <Badge variant={c.success ? "success" : "error"}>
                          {c.success ? "ok" : "erro"}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-xs" style={{ color: "var(--text-dim)", whiteSpace: "nowrap" }}>
                        {new Date(c.created_at).toLocaleString("pt-BR")}
                      </td>
                    </tr>
                  ))}
                  {(stats.recent_calls ?? []).length === 0 && (
                    <tr>
                      <td colSpan={7} className="px-4 py-8 text-center text-sm" style={{ color: "var(--text-dim)" }}>
                        Nenhuma chamada registrada ainda
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
