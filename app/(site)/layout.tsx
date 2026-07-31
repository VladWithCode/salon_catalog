import { SiteHeader } from "@/components/site/site-header";
import { SiteFooter } from "@/components/site/site-footer";
import { WhatsAppFab } from "@/components/site/whatsapp-fab";
import { getSocialLinks } from "@/lib/api/socials";

/**
 * Site chrome shared by every public marketing page.
 * Fetches social links once and hands them to the footer.
 */
export default async function SiteLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const socials = await getSocialLinks();

  return (
    <>
      <SiteHeader />
      <main id="content">{children}</main>
      <SiteFooter socials={socials} />
      <WhatsAppFab />
    </>
  );
}
