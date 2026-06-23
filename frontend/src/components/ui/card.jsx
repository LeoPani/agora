"use client";

import { cn } from "@/lib/utils";

export function Card({ children, className, style }) {
  return (
    <div
      className={cn("rounded-xl p-4", className)}
      style={{
        background: "var(--surface)",
        border: "1px solid var(--border)",
        ...style,
      }}
    >
      {children}
    </div>
  );
}

export function CardHeader({ children, className }) {
  return (
    <div className={cn("mb-3", className)}>
      {children}
    </div>
  );
}

export function CardTitle({ children, className }) {
  return (
    <h3 className={cn("text-sm font-semibold text-white", className)}>
      {children}
    </h3>
  );
}
