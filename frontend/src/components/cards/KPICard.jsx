function fmt(v) {
  if (typeof v !== "number") return v;
  return v.toLocaleString("pt-BR");
}

function OrbitalDecoration() {
  return (
    <svg className="kpi-decoration" viewBox="0 0 90 90" style={{
      position: "absolute", top: -25, right: -25, width: 90, height: 90, opacity: 0.15,
    }}>
      <circle cx="45" cy="45" r="25" fill="none" stroke="#6B21A8" strokeWidth="0.5"/>
      <g>
        <animateTransform attributeName="transform" type="rotate"
                          from="0 45 45" to="360 45 45" dur="10s" repeatCount="indefinite"/>
        <circle cx="70" cy="45" r="2.5" fill="#D4A017"/>
      </g>
    </svg>
  );
}

export function KPICard({ label, value, meta, variant = "default", icon: Icon }) {
  const isGold    = variant === "gold";
  const isOrbital = variant === "orbital";

  const bg     = isGold ? "rgba(212,160,23,0.06)"  : "var(--purple-faint)";
  const border = isGold ? "var(--gold-soft)"        : "var(--purple-soft)";
  const valCol = isGold ? "var(--gold)"             : "var(--text)";

  return (
    <div
      className="card-hover"
      style={{
        background: bg,
        border: `1px solid ${border}`,
        borderRadius: 12,
        padding: 28,
        position: "relative",
        overflow: "hidden",
      }}
    >
      {isOrbital && <OrbitalDecoration />}

      <div style={{
        color: "var(--text-dim)",
        fontSize: 11,
        letterSpacing: "0.18em",
        textTransform: "uppercase",
        marginBottom: 12,
        display: "flex",
        alignItems: "center",
        gap: 6,
      }}>
        {Icon && <Icon size={12} />}
        {label}
      </div>

      <div style={{ fontSize: 36, fontWeight: 400, letterSpacing: "-0.02em", marginBottom: 8, color: valCol }}>
        {fmt(value)}
      </div>

      {meta && (
        <div style={{ color: "var(--gold)", fontSize: 12 }}>{meta}</div>
      )}
    </div>
  );
}
