const SIZES = { sm: 40, md: 60, lg: 80 };

export function MatchingLoader({ size = "md", label }) {
  const px = SIZES[size] ?? SIZES.md;
  const c  = px / 2;
  const positions = [c * 0.3, c, c * 1.7];
  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 8 }}>
      <svg width={px} height={px} viewBox={`0 0 ${px} ${px}`}>
        {positions.map((x, i) => (
          <circle key={i} cx={x} cy={c} r="3" fill="#D4A017">
            <animate attributeName="r"       values="3;5;3" dur="1s" begin={`${i * 0.3}s`} repeatCount="indefinite"/>
            <animate attributeName="opacity" values="0.5;1;0.5" dur="1s" begin={`${i * 0.3}s`} repeatCount="indefinite"/>
          </circle>
        ))}
      </svg>
      {label && <span style={{ color: "rgba(255,255,255,0.6)", fontSize: 12 }}>{label}</span>}
    </div>
  );
}
