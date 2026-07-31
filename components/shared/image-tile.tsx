import Image from "next/image";
import { cn } from "@/lib/utils";

type ImageTileProps = {
  src: string;
  alt: string;
  /** Aspect ratio via Tailwind class, e.g. "aspect-[4/3]". Default "aspect-[4/3]". */
  ratio?: string;
  /** `sizes` attribute for responsive loading. Default "(min-width: 1024px) 33vw, 100vw". */
  sizes?: string;
  /** When true, the image gently scales on parent `group` hover. Default true. */
  hoverZoom?: boolean;
  /** Additional classes on the wrapper. */
  className?: string;
  /** Additional classes on the image itself. */
  imageClassName?: string;
  /** Priority loading (above the fold). Default false. */
  priority?: boolean;
};

/**
 * Shared image tile: `next/image` inside an overflow-hidden rounded wrapper.
 * Add `group` to a parent to enable the hover zoom.
 */
export function ImageTile({
  src,
  alt,
  ratio = "aspect-[4/3]",
  sizes = "(min-width: 1024px) 33vw, 100vw",
  hoverZoom = true,
  className,
  imageClassName,
  priority = false,
}: ImageTileProps) {
  return (
    <div className={cn("relative overflow-hidden rounded-xl", ratio, className)}>
      <Image
        src={src}
        alt={alt}
        fill
        sizes={sizes}
        priority={priority}
        className={cn(
          "object-cover",
          hoverZoom &&
            "transition-transform duration-500 ease-[cubic-bezier(0.22,1,0.36,1)] group-hover:scale-[1.04]",
          imageClassName,
        )}
      />
    </div>
  );
}
