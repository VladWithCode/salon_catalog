"use client";

import { motion, useReducedMotion } from "framer-motion";
import { type ReactNode } from "react";

type FadeInProps = {
  children: ReactNode;
  /** Delay in seconds before the animation starts. Default 0. */
  delay?: number;
  /** Duration in seconds. Default 0.5. */
  duration?: number;
  /** Translate distance on the Y axis in px. Default 12. */
  y?: number;
  className?: string;
};

/**
 * Simple opacity (and tiny upward) fade. Respects reduced motion.
 * For richer reveals use `<RevealOnView />` (scroll-triggered) or
 * `<StaggerGroup />` (parent/child orchestration).
 *
 * Always renders a `<div>`. Wrap in another element if you need a
 * `<section>` / `<li>` / etc.
 */
export function FadeIn({
  children,
  delay = 0,
  duration = 0.5,
  y = 12,
  className,
}: FadeInProps) {
  const reduced = useReducedMotion();

  if (reduced) {
    return <div className={className}>{children}</div>;
  }

  return (
    <motion.div
      className={className}
      initial={{ opacity: 0, y }}
      animate={{ opacity: 1, y: 0 }}
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
