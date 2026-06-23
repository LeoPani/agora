import "./globals.css";
import { IntroAnimation } from "@/components/intro/IntroAnimation";

export const metadata = {
  title: "Ágora · Inteligência de Inovação",
  description: "Radar de inteligência de inovação para Núcleos de Inovação Tecnológica",
  icons: {
    icon: "/favicon.svg",
  },
};

export default function RootLayout({ children }) {
  return (
    <html lang="pt-BR">
      <body>
        <IntroAnimation />
        {children}
      </body>
    </html>
  );
}
