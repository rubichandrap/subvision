'use client';

import * as React from 'react';

import { cn } from '@/lib/utils';

// A surface whose glow follows the pointer.
export function Spotlight({
  children,
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  const handleMove = (event: React.PointerEvent<HTMLDivElement>) => {
    const rect = event.currentTarget.getBoundingClientRect();
    event.currentTarget.style.setProperty('--mx', `${event.clientX - rect.left}px`);
    event.currentTarget.style.setProperty('--my', `${event.clientY - rect.top}px`);
  };
  return (
    <div onPointerMove={handleMove} className={cn('spot', className)} {...props}>
      {children}
    </div>
  );
}
