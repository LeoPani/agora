const SIZES = { sm: 40, md: 60, lg: 80 };

export function RadarLoader({ size = "md", label }) {
  const px = SIZES[size] ?? SIZES.md;
  const c  = px / 2;
  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 8 }}>
      <svg width={px} height={px} viewBox={`0 0 ${px} ${px}`}>
        <circle cx={c} cy={c} r="3" fill="#D4A017"/>
        {[0, 0.5, 1].map((delay) => (
          <circle key={delay} cx={c} cy={c} r="5" fill="none" stroke="#6B21A8" strokeWidth="1" opacity="0">
            <animate attributeName="r"       from="5" to={c * 0.83} dur="1.5s" begin={`${delay}s`} repeatCount="indefinite"/>
            <animate attributeName="opacity" from="0.8" to="0"      dur="1.5s" begin={`${delay}s`} repeatCount="indefinite"/>
          </circle>
        ))}
      </svg>
      {label && <span style={{ color: "rgba(255,255,255,0.6)", fontSize: 12 }}>{label}</span>}
    </div>
  );
}
