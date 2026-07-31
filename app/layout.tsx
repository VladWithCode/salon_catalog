import type { Metadata, Viewport } from "next";
import { Inter, Playfair_Display } from "next/font/google";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import "./globals.css";

const inter = Inter({
  subsets: ["latin", "latin-ext"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-inter",
  display: "swap",
});

const playfair = Playfair_Display({
  subsets: ["latin", "latin-ext"],
  weight: ["400", "500", "600", "700"],
  style: ["normal", "italic"],
  variable: "--font-playfair",
  display: "swap",
});

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? "https://villachenacolo.com";

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: "Villa Chenacolo · Salón de eventos de alto nivel en Durango",
    template: "%s · Villa Chenacolo",
  },
  description:
    "Salón de eventos de alto nivel en Durango. Bodas, quinceañeras, bautizos, eventos corporativos y más. Cada detalle, pensado para tu evento.",
  keywords: [
    "salón de eventos Durango",
    "Villa Chenacolo",
    "bodas Durango",
    "quinceañeras",
    "eventos corporativos",
  ],
  formatDetection: {
    telephone: false,
    email: false,
    address: false,
  },
  openGraph: {
    type: "website",
    locale: "es_MX",
    url: siteUrl,
    siteName: "Villa Chenacolo",
    title: "Villa Chenacolo · Salón de eventos de alto nivel en Durango",
    description:
      "Salón de eventos de alto nivel en Durango. Bodas, quinceañeras, bautizos, eventos corporativos y más.",
  },
  icons: {
    icon: "/favicon.ico",
  },
};

export const viewport: Viewport = {
  themeColor: "#F4ECE0",
  width: "device-width",
  initialScale: 1,
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="es" className={`${inter.variable} ${playfair.variable}`}>
      <body className="min-h-screen bg-background text-foreground antialiased">
        <a
          href="#content"
          className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[100] focus:rounded-md focus:bg-background focus:px-3 focus:py-2 focus:text-foreground focus:shadow-elevated"
        >
          Saltar al contenido
        </a>
        <TooltipProvider delayDuration={150}>{children}</TooltipProvider>
        <Toaster richColors position="top-center" closeButton />
      </body>
    </html>
  );
}
