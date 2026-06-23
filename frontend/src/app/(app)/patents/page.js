"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { FlaskConical, Search } from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

function TabBtn({ label, active, onClick }) {
  return (
    <button
      onClick={onClick}
      className="px-4 py-1.5 rounded-lg text-sm font-medium transition-all"
      style={{
        background: active ? "var(--accent)" : "var(--surface-2)",
        color: active ? "white" : "var(--text-muted)",
        border: "1px solid var(--border)",
      }}
    >
      {label}
    </button>
  );
}

export default function PatentsPage() {
  const [patents, setPatents] = useState([]);
  const [loading, setLoad]    = useState(true);
  const [q, setQ]             = useState("");
  const [tab, setTab]         = useState("all");
  const [modal, setModal]     = useState(null);

  useEffect(() => {
    fetch(`${API}/api/v1/patents?limit=500`)
      .then((r) => r.json())
      .catch(() => [])
      .then((d) => { setPatents(Array.isArray(d) ? d : d?.patents ?? []); setLoad(false); });
  }, []);

  const ufvPatents  = patents.filter((p) => p.is_ufv);
  const allPatents  = tab === "ufv" ? ufvPatents : patents;
  const filtered    = allPatents.filter((p) =>
    !q || (p.title || "").toLowerCase().includes(q.toLowerCase()) ||
    (p.inpi_number || "").toLowerCase().includes(q.toLowerCase())
  );

  // Agrupar por IPC (2 chars)
  const ipcCounts = {};
  ufvPatents.forEach((p) => {
    (p.ipc_codes || []).forEach((ipc) => {
      const k = ipc.substring(0, 1);
      ipcCounts[k] = (ipcCounts[k] || 0) + 1;
    });
  });

  const statusColor = (s) => {
    if (!s) return "muted";
    if (s.toLowerCase().includes("conced")) return "success";
    if (s.toLowerCase().includes("arquiv")) return "error";
    return "default";
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Patentes</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Portfólio de PI da UFV — INPI + Lens.org + Google Patents
        </p>
      </div>

      {/* KPIs */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[
          { label: "Total INPI",   value: patents.length,                   color: "#A78BFA" },
          { label: "UFV",          value: ufvPatents.length,                color: "#F59E0B" },
          { label: "Concedidas",   value: ufvPatents.filter((p) => (p.status||"").toLowerCase().includes("conced")).length, color: "#34D399" },
          { label: "Seções IPC",   value: Object.keys(ipcCounts).length,    color: "#60A5FA" },
        ].map(({ label, value, color }) => (
          <div key={label} className="rounded-xl p-5" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
            <p className="text-xs font-semibold uppercase tracking-widest mb-2" style={{ color: "var(--text-muted)" }}>{label}</p>
            <p className="text-3xl font-bold" style={{ color }}>{value.toLocaleString("pt-BR")}</p>
          </div>
        ))}
      </div>

      {/* Distribuição IPC */}
      {Object.keys(ipcCounts).length > 0 && (
        <div className="rounded-xl p-5" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
          <h2 className="text-sm font-semibold text-white mb-3">Distribuição por Seção IPC</h2>
          <div className="flex gap-3 flex-wrap">
            {Object.entries(ipcCounts)
              .sort((a, b) => b[1] - a[1])
              .map(([sec, cnt]) => (
                <div key={sec} className="flex items-center gap-2 px-3 py-2 rounded-lg"
                  style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}>
                  <span className="text-white font-bold text-sm">{sec}</span>
                  <span className="text-xs tabular-nums" style={{ color: "var(--text-muted)" }}>{cnt}</span>
                </div>
              ))}
          </div>
        </div>
      )}

      {/* Filtros */}
      <div className="flex gap-3 items-center flex-wrap">
        <div className="flex gap-2">
          <TabBtn label="Todas" active={tab === "all"} onClick={() => setTab("all")} />
          <TabBtn label="UFV" active={tab === "ufv"} onClick={() => setTab("ufv")} />
        </div>
        <div className="relative flex-1 min-w-48">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2" style={{ color: "var(--text-muted)" }} />
          <input
            className="w-full pl-9 pr-4 py-2 text-sm rounded-lg"
            style={{ background: "var(--surface)", border: "1px solid var(--border)", color: "var(--text)" }}
            placeholder="Buscar por título ou número INPI…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
      </div>

      {/* Tabela */}
      <div className="rounded-xl overflow-hidden" style={{ border: "1px solid var(--border)" }}>
        <table className="w-full text-sm">
          <thead>
            <tr style={{ background: "var(--surface)" }}>
              {["Nº INPI", "Título", "IPC", "Depósito", "Status"].map((h) => (
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
              <tr><td colSpan={5} className="text-center py-8" style={{ color: "var(--text-muted)" }}>
                Nenhuma patente. Execute <code>make collect-inpi ingest-inpi</code>.
              </td></tr>
            )}
            {filtered.slice(0, 100).map((p, i) => (
              <tr key={p.id || p.inpi_number || i}
                className="cursor-pointer transition-colors"
                style={{ borderBottom: "1px solid var(--border)" }}
                onClick={() => setModal(p)}
                onMouseEnter={(e) => e.currentTarget.style.background = "var(--surface-2)"}
                onMouseLeave={(e) => e.currentTarget.style.background = "transparent"}
              >
                <td className="px-4 py-3 font-mono text-xs" style={{ color: "var(--text-muted)" }}>{p.inpi_number || "—"}</td>
                <td className="px-4 py-3 text-white max-w-sm truncate">{p.title || "—"}</td>
                <td className="px-4 py-3">
                  <div className="flex gap-1 flex-wrap">
                    {(p.ipc_codes || []).slice(0, 2).map((c) => (
                      <Badge key={c} variant="muted">{c.substring(0, 4)}</Badge>
                    ))}
                  </div>
                </td>
                <td className="px-4 py-3 tabular-nums text-xs" style={{ color: "var(--text-muted)" }}>
                  {p.filing_date ? p.filing_date.substring(0, 10) : "—"}
                </td>
                <td className="px-4 py-3"><Badge variant={statusColor(p.status)}>{p.status || "em exame"}</Badge></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {filtered.length > 100 && (
        <p className="text-xs text-center" style={{ color: "var(--text-muted)" }}>
          Mostrando 100 de {filtered.length.toLocaleString("pt-BR")} resultados.
        </p>
      )}

      {/* Modal */}
      {modal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4"
          style={{ background: "rgba(0,0,0,0.7)" }} onClick={() => setModal(null)}>
          <div className="rounded-xl p-6 max-w-xl w-full max-h-[80vh] overflow-y-auto"
            style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
            onClick={(e) => e.stopPropagation()}>
            <h2 className="text-base font-bold text-white mb-3">{modal.title || modal.inpi_number}</h2>
            <div className="flex gap-2 flex-wrap mb-4">
              {modal.inpi_number && <Badge variant="muted">{modal.inpi_number}</Badge>}
              <Badge variant={statusColor(modal.status)}>{modal.status || "em exame"}</Badge>
              {modal.is_ufv && <Badge variant="gold">UFV</Badge>}
            </div>
            {modal.abstract && (
              <p className="text-sm leading-relaxed mb-4" style={{ color: "var(--text-muted)" }}>
                {modal.abstract}
              </p>
            )}
            {(modal.ipc_codes || []).length > 0 && (
              <div className="flex gap-2 flex-wrap">
                {modal.ipc_codes.map((c) => <Badge key={c} variant="default">{c}</Badge>)}
              </div>
            )}
            <button className="mt-4 text-xs px-4 py-2 rounded-lg block"
              style={{ background: "var(--surface-2)", color: "var(--text-muted)" }}
              onClick={() => setModal(null)}>Fechar</button>
          </div>
        </div>
      )}
    </div>
  );
}
