import Image from "next/image";
import Link from "next/link";

import { contact } from "@/lib/copy/contact";
import { cn } from "@/lib/utils";

type LogoProps = Readonly<{
  variant?: "light" | "dark";
  priority?: boolean;
  className?: string;
}>;

export function Logo({
  variant = "light",
  priority = false,
  className,
}: LogoProps) {
  const source =
    variant === "dark"
      ? "/assets/logo_name_white.png"
      : "/assets/logo_name.webp";

  return (
    <Link
      href="/"
      aria-label={`${contact.brand}, ir al inicio`}
      className={cn("inline-flex shrink-0 items-center", className)}
    >
      <Image
        src={source}
        alt={contact.brand}
        width={256}
        height={71}
        priority={priority}
        className="h-8 w-auto object-contain md:h-10"
      />
    </Link>
  );
}
