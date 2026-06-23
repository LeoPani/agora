import { cn } from "@/lib/utils";

const variants = {
  default: { background: "var(--accent)", color: "white" },
  muted:   { background: "var(--surface-2)", color: "var(--text-muted)" },
  gold:    { background: "#D4A01720", color: "#D4A017" },
  success: { background: "#34d39920", color: "#34d399" },
  warn:    { background: "#fbbf2420", color: "#fbbf24" },
  error:   { background: "#ef444420", color: "#f87171" },
};

export function Badge({ children, variant = "default", className }) {
  return (
    <span
      className={cn("inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium", className)}
      style={variants[variant] ?? variants.default}
    >
      {children}
    </span>
  );
}
