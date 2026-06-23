const SIZES = { sm: 40, md: 60, lg: 80 };

export function OrbitalLoader({ size = "md", label }) {
  const px = SIZES[size] ?? SIZES.md;
  const c  = px / 2;
  const r  = c * 0.67;
  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 8 }}>
      <svg width={px} height={px} viewBox={`0 0 ${px} ${px}`}>
        <circle cx={c} cy={c} r={r} fill="none" stroke="#6B21A8" strokeWidth="0.5" opacity="0.4"/>
        <circle cx={c} cy={c} r="3" fill="#6B21A8"/>
        <g>
          <animateTransform attributeName="transform" type="rotate"
                            from={`0 ${c} ${c}`} to={`360 ${c} ${c}`} dur="1.4s" repeatCount="indefinite"/>
          <circle cx={c + r} cy={c} r={px * 0.058} fill="#D4A017"/>
        </g>
      </svg>
      {label && <span style={{ color: "rgba(255,255,255,0.6)", fontSize: 12 }}>{label}</span>}
    </div>
  );
}
