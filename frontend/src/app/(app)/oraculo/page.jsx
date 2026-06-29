"use client";

import { useState, useRef, useEffect } from "react";
import { Sparkles, Send, User, Bot, ExternalLink, Plus } from "lucide-react";
import { RadarLoader } from "@/components/loaders/RadarLoader";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

const SOURCE_LABELS = {
  publication: "Publicação",
  patent: "Patente",
  opportunity: "Edital",
};

function MessageBubble({ msg }) {
  const isUser = msg.role === "user";
  return (
    <div className={`flex gap-3 ${isUser ? "flex-row-reverse" : "flex-row"}`}>
      <div
        className="w-8 h-8 rounded-full flex items-center justify-center shrink-0"
        style={{ background: isUser ? "var(--purple)" : "var(--surface-2)", border: "1px solid var(--border)" }}
      >
        {isUser ? <User size={14} color="#fff" /> : <Bot size={14} style={{ color: "var(--gold)" }} />}
      </div>

      <div style={{ maxWidth: "75%" }}>
        <div
          className="rounded-xl px-4 py-3 text-sm leading-relaxed"
          style={{
            background: isUser ? "var(--purple)" : "var(--surface)",
            border:     isUser ? "none"          : "1px solid var(--border)",
            color:      "#fff",
          }}
        >
          {msg.content}
        </div>

        {/* Sources */}
        {!isUser && msg.sources && msg.sources.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-2">
            {msg.sources.slice(0, 5).map((s) => (
              <a
                key={s.index}
                href={s.url || "#"}
                target={s.url ? "_blank" : undefined}
                rel="noreferrer"
                className="inline-flex items-center gap-1 px-2 py-1 rounded-md text-xs transition-all"
                style={{
                  background: "var(--surface-2)",
                  border:     "1px solid var(--border)",
                  color:      "var(--text-muted)",
                }}
                onMouseEnter={(e) => { e.currentTarget.style.color = "var(--gold)"; }}
                onMouseLeave={(e) => { e.currentTarget.style.color = "var(--text-muted)"; }}
              >
                <span style={{ color: "var(--gold)", fontWeight: 600 }}>[{s.index}]</span>
                <span>{SOURCE_LABELS[s.source_type] ?? s.source_type}</span>
                <span className="truncate" style={{ maxWidth: 150 }}>{s.title}</span>
                {s.url && <ExternalLink size={10} />}
              </a>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function ConvList({ conversations, activeId, onSelect, onNew }) {
  return (
    <div
      className="w-64 shrink-0 flex flex-col h-full"
      style={{ background: "var(--surface)", borderRight: "1px solid var(--border)" }}
    >
      <div className="p-4" style={{ borderBottom: "1px solid var(--border)" }}>
        <button
          onClick={onNew}
          className="w-full flex items-center justify-center gap-2 py-2 rounded-lg text-sm font-medium transition-all"
          style={{
            background: "var(--purple)",
            color:      "#fff",
            border:     "none",
          }}
          onMouseEnter={(e) => { e.currentTarget.style.background = "var(--accent-hover)"; }}
          onMouseLeave={(e) => { e.currentTarget.style.background = "var(--purple)"; }}
        >
          <Plus size={14} /> Nova conversa
        </button>
      </div>
      <div className="flex-1 overflow-y-auto py-2">
        {conversations.length === 0 && (
          <p className="text-xs px-4 py-6 text-center" style={{ color: "var(--text-dim)" }}>
            Nenhuma conversa ainda
          </p>
        )}
        {conversations.map((c) => (
          <button
            key={c.id}
            onClick={() => onSelect(c.id)}
            className="w-full text-left px-4 py-3 text-sm transition-all"
            style={{
              background: c.id === activeId ? "var(--purple-soft)" : "transparent",
              color:      c.id === activeId ? "#fff" : "var(--text-muted)",
              borderLeft: c.id === activeId ? "3px solid var(--gold)" : "3px solid transparent",
            }}
            onMouseEnter={(e) => {
              if (c.id !== activeId) e.currentTarget.style.background = "var(--purple-faint)";
            }}
            onMouseLeave={(e) => {
              if (c.id !== activeId) e.currentTarget.style.background = "transparent";
            }}
          >
            <p className="truncate font-medium" style={{ fontSize: 13 }}>{c.title}</p>
            <p style={{ fontSize: 10, color: "var(--text-dim)", marginTop: 2 }}>
              {new Date(c.updated_at).toLocaleDateString("pt-BR")}
            </p>
          </button>
        ))}
      </div>
    </div>
  );
}

export default function OraculoPage() {
  const [conversations, setConversations] = useState([]);
  const [activeConvId, setActiveConvId]   = useState(null);
  const [messages, setMessages]           = useState([]);
  const [input, setInput]                 = useState("");
  const [loading, setLoading]             = useState(false);
  const bottomRef = useRef(null);

  useEffect(() => {
    fetch(`${API}/api/conversations`)
      .then((r) => r.json())
      .catch(() => [])
      .then(setConversations);
  }, []);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  async function loadConversation(id) {
    setActiveConvId(id);
    const msgs = await fetch(`${API}/api/conversations/${id}/messages`)
      .then((r) => r.json())
      .catch(() => []);
    setMessages(Array.isArray(msgs) ? msgs.map((m) => ({
      role:    m.role,
      content: m.content,
      sources: m.sources ?? [],
    })) : []);
  }

  function startNew() {
    setActiveConvId(null);
    setMessages([]);
  }

  async function sendMessage(e) {
    e.preventDefault();
    if (!input.trim() || loading) return;

    const userMsg = input.trim();
    setInput("");
    setMessages((prev) => [...prev, { role: "user", content: userMsg, sources: [] }]);
    setLoading(true);

    try {
      const resp = await fetch(`${API}/api/chat`, {
        method:  "POST",
        headers: { "Content-Type": "application/json" },
        body:    JSON.stringify({ conversation_id: activeConvId, message: userMsg }),
      });
      const data = await resp.json();

      if (!activeConvId && data.conversation_id) {
        setActiveConvId(data.conversation_id);
        // Refresh conversation list
        fetch(`${API}/api/conversations`)
          .then((r) => r.json())
          .catch(() => [])
          .then(setConversations);
      }

      setMessages((prev) => [
        ...prev,
        { role: "assistant", content: data.message ?? data.error ?? "Erro na resposta", sources: data.sources ?? [] },
      ]);
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        { role: "assistant", content: "Erro ao contactar o servidor. Verifique se a API está rodando.", sources: [] },
      ]);
    } finally {
      setLoading(false);
    }
  }

  const suggestions = [
    "Quem pesquisa biocontrole de pragas na UFV?",
    "Quais editais estão abertos para biotecnologia?",
    "Patentes UFV na área de bioinsumos",
    "Tendências de mercado em agricultura de precisão",
  ];

  return (
    <div className="flex h-screen" style={{ marginTop: 0 }}>
      {/* Sidebar de conversas */}
      <ConvList
        conversations={conversations}
        activeId={activeConvId}
        onSelect={loadConversation}
        onNew={startNew}
      />

      {/* Área principal */}
      <div className="flex-1 flex flex-col h-full overflow-hidden">
        {/* Header */}
        <div
          className="flex items-center gap-3 px-6 py-4"
          style={{ background: "var(--surface)", borderBottom: "1px solid var(--border)" }}
        >
          <Sparkles size={18} style={{ color: "var(--gold)" }} />
          <div>
            <h1 className="text-sm font-semibold text-white">Oráculo</h1>
            <p style={{ fontSize: 11, color: "var(--text-dim)" }}>
              Chat com o data lake da UFV
            </p>
          </div>
          <div
            className="ml-auto px-2 py-0.5 rounded text-xs font-bold"
            style={{ background: "var(--gold-soft)", color: "var(--gold)", border: "1px solid rgba(212,160,23,0.3)" }}
          >
            AI
          </div>
        </div>

        {/* Mensagens */}
        <div className="flex-1 overflow-y-auto px-6 py-6 space-y-6">
          {messages.length === 0 && (
            <div className="flex flex-col items-center justify-center h-full gap-6">
              <div className="text-center">
                <Sparkles size={40} style={{ color: "var(--gold)", margin: "0 auto 12px" }} />
                <h2 className="text-white font-semibold mb-1">Oráculo do Ágora</h2>
                <p style={{ fontSize: 13, color: "var(--text-muted)", maxWidth: 400 }}>
                  Faça perguntas sobre publicações, patentes, editais, pesquisadores e oportunidades da UFV.
                </p>
              </div>
              <div className="grid grid-cols-2 gap-3" style={{ maxWidth: 480 }}>
                {suggestions.map((s) => (
                  <button
                    key={s}
                    onClick={() => setInput(s)}
                    className="text-left px-3 py-3 rounded-xl text-xs transition-all"
                    style={{
                      background: "var(--surface)",
                      border:     "1px solid var(--border)",
                      color:      "var(--text-muted)",
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.borderColor = "var(--purple)";
                      e.currentTarget.style.color = "#fff";
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.borderColor = "var(--border)";
                      e.currentTarget.style.color = "var(--text-muted)";
                    }}
                  >
                    {s}
                  </button>
                ))}
              </div>
            </div>
          )}

          {messages.map((msg, i) => (
            <MessageBubble key={i} msg={msg} />
          ))}

          {loading && (
            <div className="flex gap-3">
              <div
                className="w-8 h-8 rounded-full flex items-center justify-center shrink-0"
                style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}
              >
                <Bot size={14} style={{ color: "var(--gold)" }} />
              </div>
              <div
                className="rounded-xl px-4 py-3"
                style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
              >
                <RadarLoader size="sm" label="Consultando o data lake…" />
              </div>
            </div>
          )}

          <div ref={bottomRef} />
        </div>

        {/* Input */}
        <form
          onSubmit={sendMessage}
          className="px-6 py-4"
          style={{ background: "var(--surface)", borderTop: "1px solid var(--border)" }}
        >
          <div
            className="flex items-end gap-3 rounded-xl px-4 py-3"
            style={{ background: "var(--surface-2)", border: "1px solid var(--border)" }}
          >
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  sendMessage(e);
                }
              }}
              placeholder="Pergunte sobre publicações, patentes, editais… (Enter para enviar)"
              rows={1}
              className="flex-1 resize-none bg-transparent text-sm text-white placeholder-[var(--text-dim)] focus:outline-none"
              style={{ maxHeight: 120 }}
              disabled={loading}
            />
            <button
              type="submit"
              disabled={!input.trim() || loading}
              className="p-2 rounded-lg transition-all"
              style={{
                background:   input.trim() && !loading ? "var(--purple)" : "var(--surface)",
                color:        input.trim() && !loading ? "#fff" : "var(--text-dim)",
                border:       "1px solid var(--border)",
              }}
            >
              <Send size={15} />
            </button>
          </div>
          <p style={{ fontSize: 10, color: "var(--text-dim)", marginTop: 6, textAlign: "center" }}>
            Respostas geradas com base nos dados coletados do data lake da UFV
          </p>
        </form>
      </div>
    </div>
  );
}
