import { ContactStrip } from "@/components/site/contact-strip";
import { SiteFooter } from "@/components/site/site-footer";
import { SiteHeader } from "@/components/site/site-header";
import { WhatsAppFab } from "@/components/site/whatsapp-fab";
import { fetchCartState } from "@/lib/api/cart";
import { getSocialLinks } from "@/lib/api/socials";

export const dynamic = "force-dynamic";

export default async function SiteLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const [socials, cartResult] = await Promise.all([getSocialLinks(), fetchCartState()]);
  const cartCount = cartResult.status === "success" ? cartResult.cart.totalItems : 0;

  return (
    <>
      <SiteHeader cartCount={cartCount} />
      <main id="content" tabIndex={-1}>
        {children}
      </main>
      <ContactStrip />
      <SiteFooter socials={socials} />
      <WhatsAppFab />
    </>
  );
}
