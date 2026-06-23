"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  Database,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";

const sections = [
  {
    section: "VISÃO GERAL",
    items: [
      { href: "/",           icon: LayoutDashboard, label: "Visão Geral" },
      { href: "/collectors", icon: Database,         label: "Coletores"  },
    ],
  },
  // Futuramente:
  // { section: "RADAR", items: [
  //   { href: "/signals",  icon: Radio,    label: "Sinais"         },
  //   { href: "/matching", icon: Handshake, label: "Matchmaking"   },
  //   { href: "/editais",  icon: FileText,  label: "Editais"       },
  // ]},
  // { section: "PARCEIROS", items: [
  //   { href: "/partners", icon: Building2, label: "Empresas"      },
  // ]},
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
      className={cn(
        "fixed inset-y-0 left-0 flex flex-col z-40 transition-all duration-200",
        collapsed ? "w-14" : "w-56"
      )}
      style={{ background: "var(--surface)", borderRight: "1px solid var(--border)" }}
    >
      {/* Logo */}
      <div
        className={cn(
          "flex items-center py-5",
          collapsed ? "px-2 justify-center" : "px-5 gap-3"
        )}
        style={{ borderBottom: "1px solid var(--border)" }}
      >
        <AgoraLogo />
        {!collapsed && (
          <div className="flex-1 min-w-0">
            <p className="text-base font-bold tracking-widest text-white leading-tight">ÁGORA</p>
            <p className="text-[10px] font-medium leading-tight" style={{ color: "var(--gold)" }}>
              by Argos
            </p>
            <p className="text-[10px] mt-0.5 leading-tight" style={{ color: "var(--text-muted)" }}>
              RADAR · NIT-UFV
            </p>
          </div>
        )}
        <button
          onClick={() => setCollapsed((c) => !c)}
          title={collapsed ? "Expandir" : "Recolher"}
          className={cn(
            "p-1 rounded-md transition-all",
            collapsed && "absolute top-5 right-1"
          )}
          style={{
            color: "var(--text-muted)",
            background: "var(--surface-2)",
            border: "1px solid var(--border)",
          }}
        >
          {collapsed ? <ChevronRight size={14} /> : <ChevronLeft size={14} />}
        </button>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-2 px-2">
        {sections.map(({ section, items }) => (
          <div key={section}>
            {!collapsed && (
              <p
                className="text-[10px] font-semibold uppercase px-3 pt-4 pb-1"
                style={{ color: "var(--text-muted)", letterSpacing: "0.1em" }}
              >
                {section}
              </p>
            )}
            {items.map(({ href, icon: Icon, label }) => {
              const active = href === "/" ? path === "/" : path.startsWith(href);
              return (
                <Link
                  key={href}
                  href={href}
                  title={collapsed ? label : undefined}
                  className={cn(
                    "flex items-center gap-3 px-3 py-2.5 rounded-lg mb-0.5 text-sm transition-all",
                    collapsed && "justify-center px-0",
                    active ? "text-white font-medium" : "hover:text-white"
                  )}
                  style={{
                    color: active ? "white" : "var(--text-muted)",
                    background: active ? "var(--accent)" : "transparent",
                  }}
                  onMouseEnter={(e) => {
                    if (!active)
                      e.currentTarget.style.background = "var(--surface-2)";
                  }}
                  onMouseLeave={(e) => {
                    if (!active) e.currentTarget.style.background = "transparent";
                  }}
                >
                  <Icon size={16} className="shrink-0" />
                  {!collapsed && label}
                </Link>
              );
            })}
          </div>
        ))}
      </nav>

      {/* Footer */}
      {!collapsed && (
        <div
          className="px-4 py-3"
          style={{ borderTop: "1px solid var(--border)" }}
        >
          <p className="text-[10px] leading-relaxed" style={{ color: "var(--text-muted)" }}>
            Powered by Argos · Parceria piloto: NIT.UFV
          </p>
        </div>
      )}
    </aside>
  );
}

// Panoptes — mesmo olho do Argos, recolorido para a paleta roxa do Ágora
function AgoraLogo() {
  return (
    <svg
      width="36"
      height="36"
      viewBox="0 0 36 36"
      fill="none"
      aria-label="Ágora"
      className="shrink-0"
    >
      {/* Halo externo */}
      <circle cx="18" cy="18" r="17" stroke="#9333EA" strokeWidth="0.8" strokeOpacity="0.35" />

      {/* Olho central */}
      <ellipse cx="18" cy="18" rx="9" ry="6.5" stroke="#A855F7" strokeWidth="1.4" fill="rgba(107,33,168,0.12)" />
      <circle cx="18" cy="18" r="3.5" fill="#6B21A8" />
      <circle cx="18.8" cy="17.2" r="0.8" fill="white" opacity="0.75" />

      {/* Acento dourado — distingue do Argos */}
      <circle cx="18" cy="18" r="5" stroke="#D4A017" strokeWidth="0.5" strokeOpacity="0.4" />

      {/* Olhos secundários hexagonais */}
      <circle cx="18"  cy="4.5"  r="1.4" fill="#9333EA" opacity="0.55" />
      <circle cx="18"  cy="4.5"  r="0.4" fill="white"   opacity="0.6" />
      <circle cx="30"  cy="11"   r="1.4" fill="#9333EA" opacity="0.55" />
      <circle cx="30"  cy="11"   r="0.4" fill="white"   opacity="0.6" />
      <circle cx="30"  cy="25"   r="1.4" fill="#9333EA" opacity="0.55" />
      <circle cx="30"  cy="25"   r="0.4" fill="white"   opacity="0.6" />
      <circle cx="18"  cy="31.5" r="1.4" fill="#9333EA" opacity="0.55" />
      <circle cx="18"  cy="31.5" r="0.4" fill="white"   opacity="0.6" />
      <circle cx="6"   cy="25"   r="1.4" fill="#9333EA" opacity="0.55" />
      <circle cx="6"   cy="25"   r="0.4" fill="white"   opacity="0.6" />
      <circle cx="6"   cy="11"   r="1.4" fill="#9333EA" opacity="0.55" />
      <circle cx="6"   cy="11"   r="0.4" fill="white"   opacity="0.6" />

      {/* Linhas de rede */}
      <g stroke="#9333EA" strokeOpacity="0.2" strokeWidth="0.4">
        <line x1="18" y1="11.5" x2="18" y2="5.5" />
        <line x1="25" y1="14"   x2="29" y2="11.5" />
        <line x1="25" y1="22"   x2="29" y2="24.5" />
        <line x1="18" y1="24.5" x2="18" y2="30.5" />
        <line x1="11" y1="22"   x2="7"  y2="24.5" />
        <line x1="11" y1="14"   x2="7"  y2="11.5" />
      </g>
    </svg>
  );
}
