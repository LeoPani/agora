"use client";

import { useEffect, useState } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Database, RefreshCw } from "lucide-react";
import { formatDate } from "@/lib/utils";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

export default function ColetoresPage() {
  const [runs, setRuns] = useState(null);
  const [loading, setLoading] = useState(true);

  async function load() {
    setLoading(true);
    try {
      const r = await fetch(`${API}/api/v1/collector-runs`);
      if (r.ok) setRuns(await r.json());
    } catch {
      // API offline — mostramos estado vazio
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  const planned = [
    { name: "openalex-full",        desc: "OpenAlex · publicações UFV completas",  interval: "Mensal" },
    { name: "openalex-incremental", desc: "OpenAlex · busca incremental",           interval: "Semanal" },
    { name: "locus-dissertations",  desc: "LOCUS DSpace · teses e dissertações",    interval: "Semanal" },
    { name: "lens-citations",       desc: "Lens.org · citações PI↔Scholar",         interval: "Mensal" },
    { name: "editais-fapemig",      desc: "FAPEMIG · editais abertos",              interval: "Semanal" },
    { name: "editais-finep",        desc: "FINEP · editais abertos",                interval: "Semanal" },
  ];

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Coletores</h1>
          <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
            Status das coletas e histórico de execuções
          </p>
        </div>
        <button
          onClick={load}
          className="flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-all"
          style={{ background: "var(--surface-2)", border: "1px solid var(--border)", color: "var(--text-muted)" }}
          onMouseEnter={(e) => { e.currentTarget.style.color = "white"; }}
          onMouseLeave={(e) => { e.currentTarget.style.color = "var(--text-muted)"; }}
        >
          <RefreshCw size={13} />
          Atualizar
        </button>
      </div>

      {/* Tabela de execuções */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database size={15} style={{ color: "var(--accent)" }} />
            Histórico de execuções
          </CardTitle>
        </CardHeader>

        {loading ? (
          <div className="py-8 text-center text-sm" style={{ color: "var(--text-muted)" }}>
            Carregando…
          </div>
        ) : (runs && runs.length > 0) ? (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr style={{ borderBottom: "1px solid var(--border)" }}>
                  {["Coletor", "Início", "Término", "Status", "Registros", "Erro"].map((h) => (
                    <th
                      key={h}
                      className="text-left py-2 px-3 text-xs font-semibold"
                      style={{ color: "var(--text-muted)" }}
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {runs.map((run) => (
                  <tr
                    key={run.id}
                    style={{ borderBottom: "1px solid var(--border)" }}
                    className="hover:bg-white/5 transition-colors"
                  >
                    <td className="py-2.5 px-3 font-mono text-xs text-white">{run.collector_name}</td>
                    <td className="py-2.5 px-3 text-xs" style={{ color: "var(--text-muted)" }}>
                      {formatDate(run.started_at)}
                    </td>
                    <td className="py-2.5 px-3 text-xs" style={{ color: "var(--text-muted)" }}>
                      {run.finished_at ? formatDate(run.finished_at) : "—"}
                    </td>
                    <td className="py-2.5 px-3">
                      <Badge
                        variant={
                          run.status === "ok" ? "success" :
                          run.status === "running" ? "warn" :
                          run.status === "error" ? "error" : "muted"
                        }
                      >
                        {run.status ?? "—"}
                      </Badge>
                    </td>
                    <td className="py-2.5 px-3 text-xs text-white">
                      {(run.records_collected ?? 0).toLocaleString("pt-BR")}
                    </td>
                    <td
                      className="py-2.5 px-3 text-xs max-w-[200px] truncate"
                      style={{ color: "#f87171" }}
                      title={run.error_message}
                    >
                      {run.error_message || "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center py-10 gap-3">
            <Database size={32} style={{ color: "var(--text-muted)" }} />
            <p className="text-sm" style={{ color: "var(--text-muted)" }}>
              Nenhuma execução registrada ainda.
            </p>
            <p className="text-xs" style={{ color: "var(--text-muted)" }}>
              Execute: <code className="font-mono">make collect-openalex && make ingest-openalex</code>
            </p>
          </div>
        )}
      </Card>

      {/* Coletores planejados */}
      <Card>
        <CardHeader>
          <CardTitle>Coletores planejados</CardTitle>
          <p className="text-xs mt-0.5" style={{ color: "var(--text-muted)" }}>
            Roadmap de fontes de dados — Fase 1 e além
          </p>
        </CardHeader>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr style={{ borderBottom: "1px solid var(--border)" }}>
                {["Coletor", "Descrição", "Frequência", "Status"].map((h) => (
                  <th
                    key={h}
                    className="text-left py-2 px-3 text-xs font-semibold"
                    style={{ color: "var(--text-muted)" }}
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {planned.map((c, i) => (
                <tr key={c.name} style={{ borderBottom: "1px solid var(--border)" }}>
                  <td className="py-2.5 px-3 font-mono text-xs text-white">{c.name}</td>
                  <td className="py-2.5 px-3 text-xs" style={{ color: "var(--text-muted)" }}>{c.desc}</td>
                  <td className="py-2.5 px-3 text-xs" style={{ color: "var(--text-muted)" }}>{c.interval}</td>
                  <td className="py-2.5 px-3">
                    <Badge variant={i < 2 ? "success" : "muted"}>
                      {i < 2 ? "implementado" : "planejado"}
                    </Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <p className="text-xs text-center pb-4" style={{ color: "var(--text-muted)" }}>
        Powered by Argos · Parceria piloto: NIT.UFV
      </p>
    </div>
  );
}
