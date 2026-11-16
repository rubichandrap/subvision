'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ArrowUpRight, Captions, Github, Plus } from 'lucide-react';

import { Ambient } from '@/components/cinematic/ambient';
import { ThemeToggle } from '@/components/theme-toggle';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

const NAV = [
  { href: '/', label: 'Studio', match: (path: string) => path === '/' || path.startsWith('/editor') },
  { href: '/processes', label: 'Gallery', match: (path: string) => path.startsWith('/processes') },
];

// The shell every page shares: atmosphere, floating navigation, footer.
export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isHome = pathname === '/';

  return (
    <div className="relative flex min-h-screen flex-col">
      <Ambient />

      <header className="pointer-events-none fixed inset-x-0 top-0 z-40 flex justify-center px-3 pt-3 sm:px-6 sm:pt-4">
        <div className="glass pointer-events-auto flex h-14 w-full max-w-5xl items-center justify-between gap-2 rounded-full border border-border pl-3 pr-1.5 sm:gap-3 sm:pl-4 sm:pr-2 shadow-soft">
          <Link href="/" className="group flex items-center gap-2.5" aria-label="Subvision home">
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-brand text-primary-foreground shadow-[0_6px_18px_-6px_rgb(124_108_255/0.9)] transition-transform duration-300 group-hover:rotate-[-8deg] group-hover:scale-105">
              <Captions className="h-4 w-4" />
            </span>
            <span className="hidden text-[0.95rem] font-semibold tracking-tight min-[420px]:inline">Subvision</span>
          </Link>

          <nav className="flex items-center gap-1">
            {NAV.map((item) => {
              const active = item.match(pathname);
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  aria-current={active ? 'page' : undefined}
                  className={cn(
                    'rounded-full px-3 py-1.5 text-sm font-medium tracking-tight transition-colors duration-200 sm:px-3.5',
                    active
                      ? 'bg-foreground/10 text-foreground'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  {item.label}
                </Link>
              );
            })}
          </nav>

          <div className="flex items-center gap-1.5">
            <ThemeToggle />
            {!isHome && (
              <Button size="sm" asChild className="w-9 gap-1.5 px-0 sm:w-auto sm:px-4">
                <Link href="/" aria-label="New video">
                  <Plus className="h-4 w-4" />
                  <span className="hidden sm:inline">New video</span>
                </Link>
              </Button>
            )}
          </div>
        </div>
      </header>

      <main className="flex-1 pt-20">{children}</main>

      <footer className="relative mt-24 overflow-hidden border-t border-border">
        <div className="mx-auto flex w-full max-w-6xl flex-col items-start justify-between gap-6 px-4 pt-10 sm:flex-row sm:items-center sm:px-6">
          <p className="max-w-sm text-sm leading-relaxed text-muted-foreground">
            Reframe, caption and render on your own server. Your footage never leaves it.
          </p>
          <Link
            href="https://github.com/rubichandrap/subvision"
            target="_blank"
            rel="noreferrer"
            aria-label="Subvision on GitHub"
            className="group inline-flex items-center gap-2 rounded-full border border-border bg-foreground/[0.03] px-4 py-2 text-sm font-medium transition-colors hover:bg-foreground/[0.08]"
          >
            <Github className="h-4 w-4" />
            Source on GitHub
            <ArrowUpRight className="h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 group-hover:-translate-y-0.5 group-hover:translate-x-0.5" />
          </Link>
        </div>
        <div
          aria-hidden
          className="pointer-events-none -mb-[0.16em] mt-6 select-none whitespace-nowrap text-center text-[clamp(4.5rem,21vw,19rem)] font-semibold leading-[0.85] tracking-[-0.06em] text-transparent [-webkit-text-stroke:1px_color-mix(in_oklab,var(--foreground)_16%,transparent)] [mask-image:linear-gradient(180deg,#000_30%,transparent_92%)]"
        >
          Subvision
        </div>
      </footer>
    </div>
  );
}
