export default function SettingsPage() {
  return (
    <div className="p-6 max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Configurações</h1>
        <p className="text-sm mt-1" style={{ color: "var(--text-muted)" }}>
          Parâmetros do radar Ágora
        </p>
      </div>
      <div className="rounded-xl p-6" style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
        <p className="text-sm" style={{ color: "var(--text-muted)" }}>
          Configurações avançadas — em breve.
        </p>
      </div>
    </div>
  );
}
