const SIZES = { sm: 40, md: 60, lg: 100, xl: 200 };

export function AgoraLogo({ size = "md", animated = true, showText = false }) {
  const px = SIZES[size] ?? SIZES.md;
  const scale = px / 100;

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
      <svg
        width={px}
        height={px}
        viewBox="0 0 100 100"
        fill="none"
        aria-label="Ágora"
        style={{ flexShrink: 0 }}
      >
        <g transform="translate(50, 55)">
          {/* onda pulsante */}
          <circle r="25" fill="none" stroke="#6B21A8" strokeWidth="0.6" opacity="0">
            <animate attributeName="r"       from="25" to="50"  dur="3.5s" repeatCount="indefinite"/>
            <animate attributeName="opacity" from="0.5" to="0"  dur="3.5s" repeatCount="indefinite"/>
          </circle>

          {/* horizonte */}
          <line x1="-38" y1="14" x2="38" y2="14" stroke="#6B21A8" strokeWidth="1"/>

          {/* olho pontudo */}
          <path d="M -18 5 Q -14 -3 0 -3 Q 14 -3 18 5 Q 14 13 0 13 Q -14 13 -18 5 Z"
                fill="#1A1329" stroke="#6B21A8" strokeWidth="1.5"/>
          <circle cx="0" cy="5" r="5" fill="#D4A017"/>
          <circle cx="0" cy="5" r="2" fill="#0d0a1a"/>
          <circle cx="-1" cy="4" r="0.8" fill="white" opacity="0.7"/>

          {/* 4 colunas mini */}
          <rect x="-30" y="-12" width="5" height="26" fill="#6B21A8"/>
          <rect x="-32" y="-14" width="9" height="2"  fill="#6B21A8"/>
          <rect x="-15" y="-9"  width="5" height="23" fill="#6B21A8"/>
          <rect x="-17" y="-11" width="9" height="2"  fill="#6B21A8"/>
          <rect x="10"  y="-9"  width="5" height="23" fill="#6B21A8"/>
          <rect x="8"   y="-11" width="9" height="2"  fill="#6B21A8"/>
          <rect x="25"  y="-12" width="5" height="26" fill="#6B21A8"/>
          <rect x="23"  y="-14" width="9" height="2"  fill="#6B21A8"/>

          {/* planeta orbitando */}
          {animated && (
            <g>
              <animateTransform attributeName="transform" type="rotate"
                                from="0" to="360" dur="6s" repeatCount="indefinite"/>
              <circle cx="38" cy="0" r="2.5" fill="#D4A017"/>
            </g>
          )}
        </g>
      </svg>

      {showText && (
        <div>
          <p style={{
            fontSize: size === "xl" ? 28 : size === "lg" ? 20 : 14,
            fontWeight: 300,
            letterSpacing: "0.2em",
            color: "#ffffff",
            lineHeight: 1.1,
            margin: 0,
          }}>
            ÁGORA
          </p>
          <p style={{
            fontSize: size === "xl" ? 12 : 10,
            letterSpacing: "0.3em",
            color: "#D4A017",
            margin: "4px 0 0",
            lineHeight: 1,
          }}>
            BY ARGOS
          </p>
          {(size === "md" || size === "lg" || size === "xl") && (
            <p style={{
              fontSize: 9,
              letterSpacing: "0.2em",
              color: "rgba(255,255,255,0.4)",
              margin: "3px 0 0",
              lineHeight: 1,
            }}>
              RADAR · NIT-UFV
            </p>
          )}
        </div>
      )}
    </div>
  );
}
