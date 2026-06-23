const VARIANTS = {
  primary: {
    bg: "#6B21A8", color: "#fff", border: "none",
    hover: { bg: "#7c2cba", shadow: "0 8px 24px rgba(107,33,168,0.4)" },
  },
  gold: {
    bg: "#D4A017", color: "#0d0a1a", border: "none",
    hover: { bg: "#e5b127", shadow: "0 8px 24px rgba(212,160,23,0.4)" },
  },
  outline: {
    bg: "transparent", color: "#fff", border: "1px solid rgba(107,33,168,0.3)",
    hover: { color: "#D4A017", border: "1px solid #D4A017", bg: "rgba(212,160,23,0.05)", shadow: "none" },
  },
  ghost: {
    bg: "transparent", color: "rgba(255,255,255,0.7)", border: "none",
    hover: { bg: "rgba(107,33,168,0.1)", color: "#fff", shadow: "none" },
  },
};

const PADS = { sm: "8px 16px", md: "12px 24px", lg: "16px 32px" };
const FNT  = { sm: 12, md: 14, lg: 16 };

export function Button({ variant = "primary", size = "md", children, onClick, disabled, style, ...rest }) {
  const v = VARIANTS[variant] || VARIANTS.primary;

  function onEnter(e) {
    if (disabled) return;
    e.currentTarget.style.background  = v.hover.bg ?? v.bg;
    e.currentTarget.style.color       = v.hover.color ?? v.color;
    e.currentTarget.style.border      = v.hover.border ?? v.border ?? "none";
    e.currentTarget.style.boxShadow   = v.hover.shadow ?? "";
    e.currentTarget.style.transform   = "translateY(-1px)";
  }
  function onLeave(e) {
    e.currentTarget.style.background  = v.bg;
    e.currentTarget.style.color       = v.color;
    e.currentTarget.style.border      = v.border ?? "none";
    e.currentTarget.style.boxShadow   = "";
    e.currentTarget.style.transform   = "";
  }

  return (
    <button
      onClick={onClick}
      disabled={disabled}
      onMouseEnter={onEnter}
      onMouseLeave={onLeave}
      style={{
        background: v.bg,
        color: v.color,
        border: v.border ?? "none",
        padding: PADS[size] ?? PADS.md,
        borderRadius: 8,
        fontSize: FNT[size] ?? FNT.md,
        letterSpacing: "0.05em",
        cursor: disabled ? "not-allowed" : "pointer",
        opacity: disabled ? 0.5 : 1,
        fontFamily: "inherit",
        transition: "all 0.3s ease",
        ...style,
      }}
      {...rest}
    >
      {children}
    </button>
  );
}
