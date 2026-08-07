import type { Metadata } from "next";
import { Inter, Playfair_Display } from "next/font/google";

import "./globals.css";

const playfair = Playfair_Display({
  subsets: ["latin", "latin-ext"],
  weight: ["400", "500", "600", "700"],
  style: ["normal", "italic"],
  variable: "--font-display",
  display: "swap",
});

const inter = Inter({
  subsets: ["latin", "latin-ext"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-sans",
  display: "swap",
});

export const metadata: Metadata = {
  title: {
    default: "Villa Chenacolo",
    template: "%s · Villa Chenacolo",
  },
  description:
    "Salón de eventos de alto nivel en Durango. Bodas, quinceañeras, bautizos, eventos corporativos y más.",
  robots: {
    index: true,
    follow: true,
  },
  openGraph: {
    type: "website",
    locale: "es_MX",
    siteName: "Villa Chenacolo",
    title: "Villa Chenacolo",
    description:
      "Salón de eventos de alto nivel en Durango. Bodas, quinceañeras, bautizos, eventos corporativos y más.",
  },
  icons: {
    icon: "/assets/logo.webp",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="es" className={`${playfair.variable} ${inter.variable}`}>
      <body
        className={`${inter.className} min-h-screen bg-background text-foreground antialiased`}
      >
        <a
          href="#content"
          className="fixed top-0 left-stack z-[100] -translate-y-full rounded-md bg-primary p-stack text-primary-foreground focus-visible:translate-y-0"
        >
          Ir al contenido principal
        </a>
        {children}
      </body>
    </html>
  );
}
