import { Sidebar } from "@/components/layout/Sidebar";

export default function AppLayout({ children }) {
  return (
    <>
      <Sidebar />
      <main
        className="flex-1 min-h-screen overflow-y-auto transition-all duration-200 page-enter"
        style={{ marginLeft: "var(--sidebar-w, 14rem)" }}
      >
        {children}
      </main>
    </>
  );
}
