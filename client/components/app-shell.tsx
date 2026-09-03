'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Captions, Film, FolderOpen, Plus, Sparkles } from 'lucide-react';

import { ThemeToggle } from '@/components/theme-toggle';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

// The shell every page shares: a studio-grade header with clean navigation,
// wordmark, active states, and a minimal craft footer.
export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isHome = pathname === '/';
  const isGallery = pathname.startsWith('/processes');

  return (
    <div className="flex min-h-screen flex-col bg-background selection:bg-primary/20 selection:text-primary">
      <header className="sticky top-0 z-40 border-b border-border/70 bg-background/80 backdrop-blur-md">
        <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between gap-4 px-4 sm:px-6">
          <div className="flex items-center gap-6">
            <Link href="/" className="group flex items-center gap-2.5">
              <span className="relative flex h-8 w-8 items-center justify-center rounded-lg border border-primary/30 bg-primary/10 text-primary shadow-xs transition-colors group-hover:border-primary/50 group-hover:bg-primary/20">
                <Captions className="h-4.5 w-4.5" />
              </span>
              <div className="flex items-center gap-2">
                <span className="font-display text-base font-semibold tracking-tight sm:text-lg">
                  Subvision
                </span>
                <span className="hidden rounded-full border border-border/80 bg-muted/60 px-2 py-0.5 text-[10px] font-medium tracking-wide text-muted-foreground sm:inline-flex">
                  STUDIO
                </span>
              </div>
            </Link>

            <nav className="hidden items-center gap-1 sm:flex">
              <Link
                href="/"
                className={cn(
                  'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                  isHome
                    ? 'bg-accent/80 text-foreground font-semibold'
                    : 'text-muted-foreground hover:bg-accent/40 hover:text-foreground'
                )}
              >
                <Film className="h-3.5 w-3.5" />
                Studio
              </Link>
              <Link
                href="/processes"
                className={cn(
                  'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                  isGallery
                    ? 'bg-accent/80 text-foreground font-semibold'
                    : 'text-muted-foreground hover:bg-accent/40 hover:text-foreground'
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
              <Button
                variant={isGallery ? 'secondary' : 'ghost'}
                size="sm"
                asChild
                className="h-8 px-2.5 text-xs"
              >
                <Link href="/processes">Gallery</Link>
              </Button>
            </nav>

            <ThemeToggle />

            {!isHome && (
              <Button size="sm" asChild className="h-9 gap-1.5 px-3 text-xs sm:text-sm font-medium shadow-xs">
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
      <footer className="border-t border-border/60 bg-background/50">
        <div className="mx-auto flex w-full max-w-6xl flex-col items-center justify-between gap-3 px-4 py-6 text-center sm:flex-row sm:text-left sm:px-6">
          <div className="flex items-center gap-2">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
            <p className="text-xs text-muted-foreground">
              Subvision Studio. Video reframing and animated subtitles.
            </p>
          </div>
          <p className="font-mono text-[11px] text-muted-foreground/70">
            whisper.cpp · Remotion · ffmpeg
          </p>
        </div>
      </footer>
    </div>
  );
}
