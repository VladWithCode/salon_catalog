"use client";

import { motion, useInView, useReducedMotion } from "framer-motion";
import { useRef, type ReactNode } from "react";

type RevealOnViewProps = {
  children: ReactNode;
  /** Y translate in px the element starts at. Default 16. */
  y?: number;
  /** Delay in seconds before animating. Default 0. */
  delay?: number;
  /** Duration in seconds. Default 0.6. */
  duration?: number;
  /** When true, the animation re-plays every time the element re-enters the viewport. Default false. */
  repeat?: boolean;
  className?: string;
  /** Optional id forwarded to the underlying div. */
  id?: string;
};

/**
 * Wraps a single block of content. When the block scrolls 30% into view, it
 * fades and slides up. Animates only once (unless `repeat`).
 *
 * Respects `prefers-reduced-motion`: when set, content is rendered without
 * any motion at all.
 *
 * Always renders a `<div>`. Wrap in another element if you need a
 * `<section>` / `<li>` / etc.
 */
export function RevealOnView({
  children,
  y = 16,
  delay = 0,
  duration = 0.6,
  repeat = false,
  className,
  id,
}: RevealOnViewProps) {
  const ref = useRef<HTMLDivElement | null>(null);
  const inView = useInView(ref, { once: !repeat, amount: 0.3 });
  const reduced = useReducedMotion();

  if (reduced) {
    return (
      <div ref={ref} className={className} id={id}>
        {children}
      </div>
    );
  }

  return (
    <motion.div
      ref={ref}
      className={className}
      id={id}
      initial={{ opacity: 0, y }}
      animate={inView ? { opacity: 1, y: 0 } : { opacity: 0, y }}
      transition={{
        duration,
        delay,
        ease: [0.22, 1, 0.36, 1],
      }}
    >
      {children}
    </motion.div>
  );
}
