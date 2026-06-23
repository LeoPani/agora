"use client";

import { useEffect, useState } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  BookOpen,
  Users,
  Clock,
  CalendarClock,
  Radio,
  Zap,
} from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

export default function VisaoGeralPage() {
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch(`${API}/api/v1/stats`)
      .then((r) => r.json())
      .then((d) => { setStats(d); setLoading(false); })
      .catch(() => setLoading(false));
  }, []);

  const kpis = [
    {
      icon: BookOpen,
      label: "Publicações coletadas",
      value: loading ? "…" : (stats?.publications ?? 0).toLocaleString("pt-BR"),
      sub: "via OpenAlex",
      color: "#9333EA",
    },
    {
      icon: Users,
      label: "Pesquisadores mapeados",
      value: loading ? "…" : (stats?.researchers ?? 0).toLocaleString("pt-BR"),
      sub: "UFV, únicos",
      color: "#D4A017",
    },
    {
      icon: Clock,
      label: "Última atualização",
      value: loading ? "…" : (stats?.last_collected
        ? new Date(stats.last_collected).toLocaleDateString("pt-BR")
        : "—"),
      sub: stats?.last_collected ? "coleta concluída" : "aguardando primeira coleta",
      color: "#34d399",
    },
    {
      icon: CalendarClock,
      label: "Próxima coleta",
      value: loading ? "…" : (stats?.next_collection ?? "—"),
      sub: "refresh incremental",
      color: "#9475B4",
    },
  ];

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Visão Geral</h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Radar de Inteligência de Inovação · NIT-UFV
          </p>
        </div>
        <div className="flex items-center gap-2">
          {stats ? (
            <span className="text-xs text-emerald-400 flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse inline-block" />
              API conectada
            </span>
          ) : (
            <span className="text-xs" style={{ color: "var(--text-muted)" }}>
              {loading ? "conectando…" : "API offline"}
            </span>
          )}
        </div>
      </div>

      {/* KPI Grid */}
      <div className="grid grid-cols-4 gap-4">
        {kpis.map(({ icon: Icon, label, value, sub, color }) => (
          <Card key={label}>
            <div className="flex items-start justify-between">
              <div>
                <p className="text-xs mb-1" style={{ color: "var(--text-muted)" }}>{label}</p>
                <p className="text-2xl font-bold text-white">{value}</p>
                <p className="text-xs mt-1" style={{ color: "var(--text-muted)" }}>{sub}</p>
              </div>
              <div className="p-2 rounded-lg" style={{ background: color + "20" }}>
                <Icon size={18} style={{ color }} />
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* Placeholder sinais */}
      <Card style={{ borderColor: "#6B21A840" }}>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Radio size={15} style={{ color: "var(--accent)" }} />
            Sinais do Radar
          </CardTitle>
        </CardHeader>
        <div className="flex flex-col items-center justify-center py-12 gap-4">
          <div
            className="w-16 h-16 rounded-full flex items-center justify-center"
            style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}
          >
            <Zap size={28} style={{ color: "var(--text-muted)" }} />
          </div>
          <div className="text-center">
            <p className="text-white font-medium mb-1">
              Sinais ainda não disponíveis
            </p>
            <p className="text-sm max-w-md" style={{ color: "var(--text-muted)" }}>
              Aguardando dados suficientes para gerar matches.
              Execute a coleta OpenAlex e ingira os dados para ativar o radar.
            </p>
          </div>
          <div className="flex gap-2 mt-2">
            <Badge variant="muted">Fase 1: Coleta de dados</Badge>
            <Badge variant="muted">Fase 2: Embeddings</Badge>
            <Badge variant="muted">Fase 3: Radar ativo</Badge>
          </div>
        </div>
      </Card>

      {/* Status das fontes */}
      <Card>
        <CardHeader>
          <CardTitle>Fontes de Dados</CardTitle>
        </CardHeader>
        <div className="grid grid-cols-2 gap-3">
          {[
            { name: "OpenAlex",    desc: "Publicações UFV",      status: "ativo",     color: "#34d399" },
            { name: "LOCUS",       desc: "Teses e dissertações", status: "planejado", color: "#9475B4" },
            { name: "Lens.org",    desc: "Citações PI↔Scholar",  status: "planejado", color: "#9475B4" },
            { name: "INPI 775K",   desc: "Patentes brasileiras", status: "planejado", color: "#9475B4" },
            { name: "FAPEMIG/FINEP", desc: "Editais abertos",    status: "planejado", color: "#9475B4" },
            { name: "Comex Stat",  desc: "Importações por setor",status: "planejado", color: "#9475B4" },
          ].map((src) => (
            <div
              key={src.name}
              className="flex items-center gap-3 p-3 rounded-lg"
              style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}
            >
              <div
                className="w-2 h-2 rounded-full shrink-0"
                style={{ background: src.color }}
              />
              <div className="min-w-0">
                <p className="text-sm font-medium text-white">{src.name}</p>
                <p className="text-xs truncate" style={{ color: "var(--text-muted)" }}>{src.desc}</p>
              </div>
              <Badge
                variant={src.status === "ativo" ? "success" : "muted"}
                className="ml-auto shrink-0"
              >
                {src.status}
              </Badge>
            </div>
          ))}
        </div>
      </Card>

      {/* Footer */}
      <p className="text-xs text-center pb-4" style={{ color: "var(--text-muted)" }}>
        Powered by Argos · Parceria piloto: NIT.UFV
      </p>
    </div>
  );
}
