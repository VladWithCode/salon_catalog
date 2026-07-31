import Image from "next/image";
import Link from "next/link";
import { brand } from "@/lib/copy/contact";
import { cn } from "@/lib/utils";

type LogoProps = {
  /** "light" renders the dark wordmark (for light surfaces), "dark" renders the white wordmark (for dark surfaces). */
  variant?: "light" | "dark";
  className?: string;
  /** When true, hides the wordmark on very small screens and shows only the icon. */
  collapseOnMobile?: boolean;
};

const WORDMARK_LIGHT = "/assets/logo_name.webp";
const WORDMARK_DARK = "/assets/logo_name_white.png";
const ICON = "/assets/logo.webp";

/**
 * Villa Chenacolo wordmark, linked to the home page.
 */
export function Logo({ variant = "light", className, collapseOnMobile = false }: LogoProps) {
  const src = variant === "light" ? WORDMARK_LIGHT : WORDMARK_DARK;
  return (
    <Link
      href="/"
      aria-label={`${brand.name} — inicio`}
      className={cn("flex items-center gap-3 shrink-0", className)}
    >
      {collapseOnMobile && (
        <Image
          src={ICON}
          alt=""
          width={512}
          height={512}
          className="h-9 w-9 md:hidden"
          priority
        />
      )}
      <Image
        src={src}
        alt={brand.name}
        width={512}
        height={141}
        className={cn(
          "h-8 w-auto md:h-10",
          collapseOnMobile && "hidden md:block",
        )}
        priority
      />
    </Link>
  );
}
