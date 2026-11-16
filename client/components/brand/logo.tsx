import * as React from 'react';

import { cn } from '@/lib/utils';

// The Subvision mark: a viewfinder (two opposing frame corners) holding two
// caption lines that step down the same diagonal. The upper line is the word
// being spoken, lit; the lower is the next one, dimmed. Stepped
// like that, the lines also hint at an "S". Brackets and the dim line follow the text colour, so the
// mark works on any surface; the lit line is always the brand gradient.
// Standalone files for other uses live in /public/logo.

export function LogoMark({ className, ...props }: React.SVGProps<SVGSVGElement>) {
  const gradient = React.useId();
  return (
    <svg
      viewBox="0 0 64 64"
      fill="none"
      aria-hidden
      className={cn('shrink-0', className)}
      {...props}
    >
      <defs>
        <linearGradient id={gradient} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#8a7bff" />
          <stop offset="1" stopColor="#4c8dff" />
        </linearGradient>
      </defs>
      <path
        d="M8 24V8h16M56 40v16H40"
        stroke="currentColor"
        strokeWidth="6"
        strokeLinejoin="miter"
      />
      <rect x="14" y="22.5" width="26" height="8" rx="4" fill={`url(#${gradient})`} />
      <rect x="24" y="33.5" width="26" height="8" rx="4" fill="currentColor" opacity=".45" />
    </svg>
  );
}

// "sub" in a light, receding weight and "vision" in a heavy one: the
// subtitle underneath, the picture on top.
export function Wordmark({ className }: { className?: string }) {
  return (
    <span className={cn('tracking-[-0.035em]', className)}>
      <span className="font-normal opacity-60">sub</span>
      <span className="font-semibold">vision</span>
    </span>
  );
}

export function Logo({ className, markClassName }: { className?: string; markClassName?: string }) {
  return (
    <span className={cn('inline-flex items-center gap-2.5', className)}>
      <LogoMark className={cn('h-8 w-8', markClassName)} />
      <Wordmark className="text-[1.15rem] leading-none" />
    </span>
  );
}
