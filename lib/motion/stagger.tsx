"use client";

import { motion, useInView, useReducedMotion } from "framer-motion";
import { createContext, useContext, useRef, type ReactNode } from "react";

type StaggerGroupProps = {
  children: ReactNode;
  /** Gap between each child animation in seconds. Default 0.08. */
  staggerStep?: number;
  /** Delay before the first child animates in seconds. Default 0. */
  delay?: number;
  /** Y translate for each child in px. Default 16. */
  y?: number;
  /** Per-child duration in seconds. Default 0.5. */
  duration?: number;
  className?: string;
  /** Replay the animation every time the group re-enters the viewport. Default false. */
  repeat?: boolean;
};

type StaggerItemProps = {
  children: ReactNode;
  className?: string;
};

const StaggerContext = createContext<{ y: number; duration: number }>({
  y: 16,
  duration: 0.5,
});

/**
 * Parent container that staggers the entrance of its `<StaggerItem />`
 * children once 20% of the group is in view. Respects reduced motion.
 */
export function StaggerGroup({
  children,
  staggerStep = 0.08,
  delay = 0,
  y = 16,
  duration = 0.5,
  className,
  repeat = false,
}: StaggerGroupProps) {
  const ref = useRef<HTMLDivElement | null>(null);
  const inView = useInView(ref, { once: !repeat, amount: 0.2 });
  const reduced = useReducedMotion();

  if (reduced) {
    return <div className={className}>{children}</div>;
  }

  return (
    <motion.div
      ref={ref}
      className={className}
      initial="hidden"
      animate={inView ? "visible" : "hidden"}
      variants={{
        hidden: {},
        visible: {
          transition: {
            staggerChildren: staggerStep,
            delayChildren: delay,
          },
        },
      }}
    >
      <StaggerContext.Provider value={{ y, duration }}>
        {children}
      </StaggerContext.Provider>
    </motion.div>
  );
}

/** A child of `<StaggerGroup />` — fades and slides up in turn. */
export function StaggerItem({ children, className }: StaggerItemProps) {
  const { y, duration } = useContext(StaggerContext);
  return (
    <motion.div
      className={className}
      variants={{
        hidden: { opacity: 0, y },
        visible: {
          opacity: 1,
          y: 0,
          transition: {
            duration,
            ease: [0.22, 1, 0.36, 1],
          },
        },
      }}
    >
      {children}
    </motion.div>
  );
}

/**
 * Convenience: turns an array of items into a list of `<StaggerItem />`s.
 * Skips the boilerplate of mapping in the consumer. Each item gets a stable
 * `key` derived from its index — the `render` callback can override this by
 * passing its own `key` on the returned element.
 */
export function StaggerList<T>({
  items,
  className,
  itemClassName,
  render,
  staggerStep,
  delay,
  y,
  duration,
  repeat,
}: {
  items: readonly T[];
  className?: string;
  itemClassName?: string;
  render: (item: T, index: number) => ReactNode;
  staggerStep?: number;
  delay?: number;
  y?: number;
  duration?: number;
  repeat?: boolean;
}) {
  return (
    <StaggerGroup
      className={className}
      staggerStep={staggerStep}
      delay={delay}
      y={y}
      duration={duration}
      repeat={repeat}
    >
      {items.map((item, i) => (
        <StaggerItem key={i} className={itemClassName}>
          {render(item, i)}
        </StaggerItem>
      ))}
    </StaggerGroup>
  );
}
