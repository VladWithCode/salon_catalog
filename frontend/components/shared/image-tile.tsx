import Image from "next/image";
import Link from "next/link";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";
import type { ImageAsset } from "@/lib/types";

type ImageTileProps = Readonly<{
  image: ImageAsset;
  sizes: string;
  href?: string;
  children?: ReactNode;
  className?: string;
  imageClassName?: string;
  priority?: boolean;
}>;

export function ImageTile({
  image,
  sizes,
  href,
  children,
  className,
  imageClassName,
  priority = false,
}: ImageTileProps) {
  const content = (
    <>
      <Image
        src={image.src}
        alt={image.alt}
        fill
        sizes={sizes}
        priority={priority}
        className={cn(
          "object-cover transition-transform duration-500 ease-elegant group-hover:scale-[1.04]",
          imageClassName,
        )}
      />
      {children}
    </>
  );

  const classes = cn(
    "group relative block overflow-hidden rounded-xl shadow-card",
    className,
  );

  return href ? (
    <Link href={href} prefetch={false} className={classes}>
      {content}
    </Link>
  ) : (
    <div className={classes}>{content}</div>
  );
}
