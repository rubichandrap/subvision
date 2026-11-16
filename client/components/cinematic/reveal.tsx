'use client';

import * as React from 'react';

import { cn } from '@/lib/utils';

// Fades a block up once it scrolls into view (CSS in globals.css handles the
// motion and respects prefers-reduced-motion).
export function Reveal({
  children,
  className,
  delay = 0,
}: {
  children: React.ReactNode;
  className?: string;
  delay?: number;
}) {
  const ref = React.useRef<HTMLDivElement>(null);
  // Content is visible by default (server render, no JS, reduced motion).
  // Only blocks that start below the fold are armed to animate in.
  const [state, setState] = React.useState<'visible' | 'armed' | 'in'>('visible');

  React.useEffect(() => {
    const node = ref.current;
    if (!node || typeof IntersectionObserver === 'undefined') return;
    if (node.getBoundingClientRect().top < window.innerHeight * 0.92) return;
    setState('armed');
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setState('in');
          observer.disconnect();
        }
      },
      { threshold: 0.12, rootMargin: '0px 0px -6% 0px' }
    );
    observer.observe(node);
    // Safety net: never leave content hidden if the observer is starved.
    const fallback = window.setTimeout(() => setState('in'), 4000);
    return () => {
      observer.disconnect();
      window.clearTimeout(fallback);
    };
  }, []);

  return (
    <div
      ref={ref}
      {...(state !== 'visible' ? { 'data-reveal': '', 'data-in': state === 'in' } : {})}
      style={{ '--reveal-delay': `${delay}ms` } as React.CSSProperties}
      className={cn(className)}
    >
      {children}
    </div>
  );
}
