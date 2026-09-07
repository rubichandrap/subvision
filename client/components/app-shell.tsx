'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Captions, Film, FolderOpen, Github, Plus } from 'lucide-react';

import { ThemeToggle } from '@/components/theme-toggle';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

// The shell every page shares: header, nav, active states, and footer.
export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isHome = pathname === '/';
  const isGallery = pathname.startsWith('/processes');

  return (
    <div className="flex min-h-screen flex-col bg-background selection:bg-primary selection:text-primary-foreground">
      <header className="sticky top-0 z-40 border-b-2 border-border bg-background">
        <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between gap-4 px-4 sm:px-6">
          <div className="flex items-center gap-6">
            <Link href="/" className="group flex items-center gap-2.5">
              <span className="flex h-9 w-9 items-center justify-center border-2 border-border bg-primary text-primary-foreground shadow-brutal-sm transition-all duration-100 group-hover:-translate-x-0.5 group-hover:-translate-y-0.5 group-hover:shadow-brutal">
                <Captions className="h-4.5 w-4.5" />
              </span>
              <span className="font-display text-base font-semibold tracking-tight sm:text-lg">
                Subvision
              </span>
            </Link>

            <nav className="hidden items-center gap-2 sm:flex">
              <Link
                href="/"
                className={cn(
                  'inline-flex items-center gap-1.5 border-2 px-3 py-1.5 font-mono text-xs font-bold uppercase tracking-wide transition-all duration-100',
                  isHome
                    ? 'border-border bg-primary text-primary-foreground shadow-brutal-sm'
                    : 'border-transparent text-muted-foreground hover:border-border hover:bg-accent hover:text-accent-foreground'
                )}
              >
                <Film className="h-3.5 w-3.5" />
                Home
              </Link>
              <Link
                href="/processes"
                className={cn(
                  'inline-flex items-center gap-1.5 border-2 px-3 py-1.5 font-mono text-xs font-bold uppercase tracking-wide transition-all duration-100',
                  isGallery
                    ? 'border-border bg-primary text-primary-foreground shadow-brutal-sm'
                    : 'border-transparent text-muted-foreground hover:border-border hover:bg-accent hover:text-accent-foreground'
                )}
              >
                <FolderOpen className="h-3.5 w-3.5" />
                Gallery
              </Link>
            </nav>
          </div>

          <div className="flex items-center gap-2">
            {/* Mobile nav links */}
            <nav className="flex items-center gap-1 sm:hidden">
              <Button size="sm" asChild className="h-9 gap-1.5 border-2 px-3 text-xs sm:text-sm font-bold shadow-brutal-sm">
                <Link href="/processes">Gallery</Link>
              </Button>
            </nav>

            <ThemeToggle />

            {!isHome && (
              <Button size="sm" asChild className="h-9 gap-1.5 px-3 text-xs sm:text-sm font-bold">
                <Link href="/">
                  <Plus className="h-3.5 w-3.5" />
                  <span>New video</span>
                </Link>
              </Button>
            )}
          </div>
        </div>
      </header>
      <main className="flex-1">{children}</main>
      <footer className="border-t-2 border-border bg-muted/40">
        <div className="mx-auto flex w-full max-w-6xl flex-col items-center justify-between gap-3 px-4 py-6 text-center sm:flex-row sm:text-left sm:px-6">
          <p className="text-xs text-muted-foreground">
            Subvision. Video reframing and animated subtitles.
          </p>
          <Link
            href="https://github.com/rubichandrap/subvision"
            target="_blank"
            rel="noreferrer"
            aria-label="Subvision on GitHub"
            className="inline-flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            <Github className="h-4 w-4" />
            GitHub
          </Link>
        </div>
      </footer>
    </div>
  );
}
