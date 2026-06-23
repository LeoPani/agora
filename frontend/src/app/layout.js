import "./globals.css";

export const metadata = {
  title: "Ágora by Argos",
  description: "Radar de Inteligência de Inovação · NIT-UFV",
};

export default function RootLayout({ children }) {
  return (
    <html lang="pt-BR">
      <body>{children}</body>
    </html>
  );
}
