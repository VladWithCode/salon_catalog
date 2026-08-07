import type { SocialLink } from "@/lib/types";

type SocialIconsProps = Readonly<{
  socials: readonly SocialLink[];
}>;

type SocialIconProps = Readonly<{
  className?: string;
}>;

function FacebookIcon({ className }: SocialIconProps) {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" className={className}>
      <path
        fill="currentColor"
        d="M13.5 21v-8h2.75l.41-3.2H13.5V7.76c0-.93.26-1.56 1.59-1.56h1.7V3.34c-.29-.04-1.3-.13-2.47-.13-2.45 0-4.13 1.5-4.13 4.25V9.8H7.42V13h2.77v8h3.31Z"
      />
    </svg>
  );
}

function InstagramIcon({ className }: SocialIconProps) {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" className={className}>
      <path
        fill="currentColor"
        d="M12 2.16c3.2 0 3.58.02 4.85.07 3.25.15 4.77 1.69 4.92 4.92.06 1.27.07 1.65.07 4.85s-.01 3.58-.07 4.85c-.15 3.23-1.67 4.77-4.92 4.92-1.27.06-1.65.07-4.85.07s-3.58-.01-4.85-.07c-3.26-.15-4.77-1.7-4.92-4.92-.06-1.27-.07-1.65-.07-4.85s.01-3.58.07-4.85c.15-3.23 1.66-4.77 4.92-4.92C8.42 2.18 8.8 2.16 12 2.16Zm0 2.17c-3.15 0-3.52.01-4.75.07-2.17.1-3.18 1.12-3.28 3.28-.06 1.23-.07 1.6-.07 4.75s.01 3.52.07 4.75c.1 2.16 1.11 3.18 3.28 3.28 1.23.06 1.6.07 4.75.07s3.52-.01 4.75-.07c2.17-.1 3.18-1.12 3.28-3.28.06-1.23.07-1.6.07-4.75s-.01-3.52-.07-4.75c-.1-2.16-1.11-3.18-3.28-3.28-1.23-.06-1.6-.07-4.75-.07Zm0 3.68A4.42 4.42 0 1 1 12 16.85 4.42 4.42 0 0 1 12 8.01Zm0 7.29a2.87 2.87 0 1 0 0-5.74 2.87 2.87 0 0 0 0 5.74Zm4.66-8.5a1.03 1.03 0 1 1 0 2.06 1.03 1.03 0 0 1 0-2.06Z"
      />
    </svg>
  );
}

function WhatsAppIcon({ className }: SocialIconProps) {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" className={className}>
      <path
        fill="currentColor"
        d="M12.04 2a9.84 9.84 0 0 0-8.45 14.87L2 22l5.27-1.55A9.93 9.93 0 1 0 12.04 2Zm0 17.98a8.1 8.1 0 0 1-4.13-1.13l-.3-.18-3.13.92.94-3.05-.2-.31a8.08 8.08 0 1 1 6.82 3.75Zm4.43-6.06c-.24-.12-1.44-.71-1.66-.79-.22-.08-.38-.12-.55.12-.16.24-.62.79-.76.95-.14.16-.28.18-.52.06-.24-.12-1.02-.38-1.94-1.2a7.27 7.27 0 0 1-1.34-1.67c-.14-.24-.01-.37.11-.49.11-.11.24-.28.36-.42.12-.14.16-.24.24-.4.08-.16.04-.3-.02-.42-.06-.12-.55-1.31-.75-1.8-.2-.47-.4-.4-.55-.41h-.46c-.16 0-.42.06-.64.3-.22.24-.84.82-.84 2s.86 2.32.98 2.48c.12.16 1.69 2.58 4.1 3.62.57.25 1.02.39 1.37.5.58.18 1.1.16 1.51.1.46-.07 1.44-.59 1.64-1.16.2-.57.2-1.06.14-1.16-.06-.1-.22-.16-.46-.28Z"
      />
    </svg>
  );
}

function iconFor(name: string) {
  const normalizedName = name.toLocaleLowerCase("es");
  if (normalizedName.includes("facebook")) return FacebookIcon;
  if (normalizedName.includes("instagram")) return InstagramIcon;
  if (normalizedName.includes("whatsapp")) return WhatsAppIcon;
  return null;
}

export function SocialIcons({ socials }: SocialIconsProps) {
  const supportedSocials = socials.flatMap((social) => {
    const Icon = iconFor(social.name);
    return Icon ? [{ social, Icon }] : [];
  });

  if (supportedSocials.length === 0) {
    return null;
  }

  return (
    <ul aria-label="Redes sociales" className="flex flex-wrap gap-2">
      {supportedSocials.map(({ social, Icon }) => (
        <li key={`${social.name}-${social.link}`}>
          <a
            href={social.link}
            target="_blank"
            rel="noopener noreferrer"
            aria-label={`Visitar ${social.name} de Villa Chenacolo`}
            className="inline-flex size-11 items-center justify-center rounded-full border border-primary-foreground/20 text-primary-foreground/75 hover:border-accent hover:text-accent-on-dark"
          >
            <Icon className="size-5" />
          </a>
        </li>
      ))}
    </ul>
  );
}
