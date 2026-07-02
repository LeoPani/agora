"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { 
  Lightbulb, Building2, Globe, Shield, TrendingUp, Users, ChevronDown, ChevronUp, CheckCircle
} from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

const SIGNAL_TYPES = {
  research_potential: { icon: Lightbulb, color: "#A78BFA", label: "Potencial de PI" },
  researcher_company_match: { icon: Building2, color: "#34D399", label: "Match de Mercado" },
  import_gap: { icon: Globe, color: "#FBBF24", label: "Gap de Importação" },
  patent_pool: { icon: Shield, color: "#FB7185", label: "Pool de Patentes" },
  trend_window: { icon: TrendingUp, color: "#C084FC", label: "Janela de Tendência" },
  interdept_collab: { icon: Users, color: "#60A5FA", label: "Colaboração Inédita" }
};

function SignalCard({ sig, onReview }) {
  const [expanded, setExpanded] = useState(false);
  const typeInfo = SIGNAL_TYPES[sig.signal_type] || { icon: Lightbulb, color: "#9CA3AF", label: sig.signal_type };
  const Icon = typeInfo.icon;
  
  const relevance = sig.relevance || (sig.score >= 0.7 ? "Alta" : sig.score >= 0.4 ? "Média" : "Baixa");
  const relColor = relevance === "Alta" ? "success" : relevance === "Média" ? "gold" : "muted";

  return (
    <div 
      className="rounded-xl p-5 flex flex-col gap-3 transition-transform hover:-translate-y-[2px]"
      style={{ 
        background: "var(--surface)", 
        border: "1px solid var(--border)",
        boxShadow: "0 4px 20px rgba(0,0,0,0.2)"
      }}
    >
      <div className="flex items-center gap-3">
        <div className="p-2 rounded-lg" style={{ background: typeInfo.color + "20" }}>
          <Icon size={18} style={{ color: typeInfo.color }} />
        </div>
        <div className="flex-1">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold uppercase tracking-wider" style={{ color: typeInfo.color }}>
              {typeInfo.label}
            </span>
            <Badge variant={relColor}>{relevance}</Badge>
          </div>
          <h3 className="text-base font-bold text-white mt-1 leading-snug">{sig.title}</h3>
        </div>
      </div>
      
      <div className="flex items-center justify-between mt-2">
        <span className="text-sm font-medium" style={{ color: "var(--text-muted)" }}>Score: {(sig.score * 100).toFixed(0)}%</span>
        <button 
          onClick={() => setExpanded(!expanded)}
          className="text-xs flex items-center gap-1 hover:text-white transition-colors"
          style={{ color: "var(--text-muted)" }}
        >
          {expanded ? "Ocultar detalhes" : "Ver detalhes"}
          {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
        </button>
      </div>

      {expanded && (
        <div className="mt-3 pt-3 space-y-3" style={{ borderTop: "1px solid var(--border)" }}>
          <p className="text-sm leading-relaxed text-gray-300">
            {sig.description}
          </p>
          {sig.reasoning && (
            <div className="p-3 rounded-md bg-black/20 text-xs text-gray-400">
              <span className="font-semibold text-gray-300 block mb-1">Análise:</span>
              {JSON.stringify(sig.reasoning)}
            </div>
          )}
          {sig.entities && (
            <div className="p-3 rounded-md bg-black/20 text-xs text-gray-400 overflow-hidden text-ellipsis">
              <span className="font-semibold text-gray-300 block mb-1">Entidades Envolvidas:</span>
              {JSON.stringify(sig.entities)}
            </div>
          )}
          
          <button 
            onClick={() => onReview(sig.id)}
            className="w-full flex items-center justify-center gap-2 py-2 mt-2 rounded-lg transition-colors"
            style={{ background: "var(--accent-hover)", color: "white" }}
          >
            <CheckCircle size={16} />
            Marcar como revisado
          </button>
        </div>
      )}
    </div>
  );
}

export default function SignalsPage() {
  const [signals, setSignals] = useState([]);
  const [loading, setLoad] = useState(true);

  const loadSignals = () => {
    fetch(`${API}/api/v1/signals?limit=50`)
      .then(r => r.json())
      .catch(() => [])
      .then(d => {
        setSignals(Array.isArray(d) ? d : []);
        setLoad(false);
      });
  };

  useEffect(() => {
    loadSignals();
  }, []);

  const handleReview = async (id) => {
    try {
      await fetch(`${API}/api/v1/signals/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status: "reviewed" })
      });
      loadSignals();
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Sinais do Radar</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Inteligência acionável e conexões descobertas automaticamente.
        </p>
      </div>

      {loading && (
        <div className="py-16 text-center" style={{ color: "var(--text-muted)" }}>Analisando radar…</div>
      )}
      {!loading && signals.length === 0 && (
        <div className="py-16 text-center" style={{ color: "var(--text-muted)" }}>
          Nenhum sinal ativo no momento. Execute <code>make generate-signals</code>.
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {signals.filter(s => s.status !== "reviewed").map(s => (
          <SignalCard key={s.id} sig={s} onReview={handleReview} />
        ))}
      </div>
    </div>
  );
}
