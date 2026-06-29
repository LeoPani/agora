"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  BookOpen,
  FlaskConical,
  Users,
  FileText,
  BarChart3,
  TrendingUp,
  Handshake,
  Database,
  Settings,
  ChevronLeft,
  ChevronRight,
  Sparkles,
  Bot,
  Activity,
} from "lucide-react";

function AIBadge() {
  return (
    <span
      style={{
        fontSize:    8,
        fontWeight:  700,
        lineHeight:  1,
        padding:     "2px 4px",
        borderRadius: 4,
        background:  "rgba(212,160,23,0.2)",
        color:       "#D4A017",
        border:      "1px solid rgba(212,160,23,0.3)",
        marginLeft:  4,
        verticalAlign: "middle",
      }}
    >
      AI
    </span>
  );
}

const sections = [
  {
    section: "EXPLORAR",
    items: [
      { href: "/",              icon: LayoutDashboard, label: "Visão Geral"    },
      { href: "/publications",  icon: BookOpen,        label: "Publicações"    },
      { href: "/patents",       icon: FlaskConical,    label: "Patentes"       },
      { href: "/groups",        icon: Users,           label: "Grupos"         },
      { href: "/opportunities", icon: FileText,        label: "Oportunidades"  },
      { href: "/imports",       icon: BarChart3,       label: "Importações"    },
      { href: "/trends",        icon: TrendingUp,      label: "Tendências"     },
      { href: "/partners",      icon: Handshake,       label: "Interessados"   },
    ],
  },
  {
    section: "INTELIGÊNCIA",
    items: [
      { href: "/oraculo", icon: Sparkles, label: "Oráculo", badge: true },
      { href: "/agente",  icon: Bot,      label: "Agente",  badge: true },
    ],
  },
  {
    section: "SISTEMA",
    items: [
      { href: "/collectors",        icon: Database,  label: "Coletores"     },
      { href: "/sistema/llm-stats", icon: Activity,  label: "LLM Stats"     },
      { href: "/settings",          icon: Settings,  label: "Configurações" },
    ],
  },
];

export function Sidebar() {
  const path = usePathname();
  const [collapsed, setCollapsed] = useState(false);

  useEffect(() => {
    document.documentElement.style.setProperty(
      "--sidebar-w",
      collapsed ? "3.5rem" : "14rem"
    );
  }, [collapsed]);

  return (
    <aside
      className={cn("fixed inset-y-0 left-0 flex flex-col z-40 transition-all duration-200",
        collapsed ? "w-14" : "w-56")}
      style={{ background: "var(--surface)", borderRight: "1px solid var(--border)" }}
    >
      {/* Logo */}
      <div
        className={cn("flex items-center py-4", collapsed ? "px-2 justify-center" : "px-4 gap-3")}
        style={{ borderBottom: "1px solid var(--border)" }}
      >
        <AgoraLogoMini animated={!collapsed} />
        {!collapsed && (
          <div className="flex-1 min-w-0">
            <p style={{ fontSize: 14, fontWeight: 300, letterSpacing: "0.2em", color: "#fff", lineHeight: 1.1, margin: 0 }}>
              ÁGORA
            </p>
            <p style={{ fontSize: 9, letterSpacing: "0.3em", color: "#D4A017", margin: "3px 0 0", lineHeight: 1 }}>
              BY ARGOS
            </p>
            <p style={{ fontSize: 8, letterSpacing: "0.15em", color: "var(--text-dim)", margin: "2px 0 0", lineHeight: 1 }}>
              RADAR · NIT-UFV
            </p>
          </div>
        )}
        <button
          onClick={() => setCollapsed((c) => !c)}
          title={collapsed ? "Expandir" : "Recolher"}
          className={cn("p-1 rounded-md transition-all", collapsed && "absolute top-4 right-1")}
          style={{
            color: "var(--text-muted)",
            background: "var(--surface-2)",
            border: "1px solid var(--border)",
          }}
        >
          {collapsed ? <ChevronRight size={13} /> : <ChevronLeft size={13} />}
        </button>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-2 px-2">
        {sections.map(({ section, items }) => (
          <div key={section}>
            {!collapsed && (
              <p className="text-[9px] font-semibold uppercase px-3 pt-4 pb-1"
                 style={{ color: "var(--text-dim)", letterSpacing: "0.15em" }}>
                {section}
              </p>
            )}
            {items.map(({ href, icon: Icon, label, badge }) => {
              const active = href === "/" ? path === "/" : path.startsWith(href);
              return (
                <Link
                  key={href}
                  href={href}
                  title={collapsed ? label : undefined}
                  className={cn(
                    "flex items-center gap-3 py-2.5 rounded-lg mb-0.5 text-sm transition-all group",
                    collapsed ? "justify-center px-0" : "px-3"
                  )}
                  style={{
                    color:      active ? "#fff" : "var(--text-muted)",
                    background: active ? "var(--purple-soft)" : "transparent",
                    borderLeft: active && !collapsed ? "3px solid var(--gold)" : "3px solid transparent",
                    paddingLeft: active && !collapsed ? "calc(0.75rem - 3px)" : undefined,
                    fontWeight: active ? 500 : 400,
                  }}
                  onMouseEnter={(e) => {
                    if (!active) {
                      e.currentTarget.style.background = "var(--purple-faint)";
                      e.currentTarget.style.color = "#fff";
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (!active) {
                      e.currentTarget.style.background = "transparent";
                      e.currentTarget.style.color = "var(--text-muted)";
                    }
                  }}
                >
                  <Icon
                    size={15}
                    className="shrink-0 transition-transform duration-200 group-hover:scale-110"
                    style={{ color: active ? "var(--gold)" : "inherit" }}
                  />
                  {!collapsed && (
                    <span className="flex items-center">
                      {label}
                      {badge && <AIBadge />}
                    </span>
                  )}
                </Link>
              );
            })}
          </div>
        ))}
      </nav>

      {/* Footer */}
      {!collapsed && (
        <div className="px-4 py-3" style={{ borderTop: "1px solid var(--border)" }}>
          <p style={{ fontSize: 10, color: "var(--text-dim)", lineHeight: 1.5 }}>
            Powered by Argos
          </p>
          <p style={{ fontSize: 9, color: "rgba(255,255,255,0.2)", marginTop: 2 }}>v0.1.0</p>
        </div>
      )}
    </aside>
  );
}

function AgoraLogoMini({ animated = true }) {
  return (
    <svg width="32" height="32" viewBox="0 0 100 100" fill="none" aria-label="Ágora" style={{ flexShrink: 0 }}>
      <g transform="translate(50, 55)">
        <circle r="25" fill="none" stroke="#6B21A8" strokeWidth="0.6" opacity="0">
          <animate attributeName="r"       from="25" to="50" dur="3.5s" repeatCount="indefinite"/>
          <animate attributeName="opacity" from="0.5" to="0" dur="3.5s" repeatCount="indefinite"/>
        </circle>
        <line x1="-38" y1="14" x2="38" y2="14" stroke="#6B21A8" strokeWidth="1"/>
        <path d="M -18 5 Q -14 -3 0 -3 Q 14 -3 18 5 Q 14 13 0 13 Q -14 13 -18 5 Z"
              fill="#1A1329" stroke="#6B21A8" strokeWidth="1.5"/>
        <circle cx="0" cy="5" r="5" fill="#D4A017"/>
        <circle cx="0" cy="5" r="2" fill="#0d0a1a"/>
        <rect x="-30" y="-12" width="5" height="26" fill="#6B21A8"/>
        <rect x="-32" y="-14" width="9" height="2"  fill="#6B21A8"/>
        <rect x="-15" y="-9"  width="5" height="23" fill="#6B21A8"/>
        <rect x="-17" y="-11" width="9" height="2"  fill="#6B21A8"/>
        <rect x="10"  y="-9"  width="5" height="23" fill="#6B21A8"/>
        <rect x="8"   y="-11" width="9" height="2"  fill="#6B21A8"/>
        <rect x="25"  y="-12" width="5" height="26" fill="#6B21A8"/>
        <rect x="23"  y="-14" width="9" height="2"  fill="#6B21A8"/>
        {animated && (
          <g>
            <animateTransform attributeName="transform" type="rotate"
                              from="0" to="360" dur="6s" repeatCount="indefinite"/>
            <circle cx="38" cy="0" r="2.5" fill="#D4A017"/>
          </g>
        )}
      </g>
    </svg>
  );
}
