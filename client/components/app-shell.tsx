import Link from 'next/link';
import { Captions } from 'lucide-react';

import { ThemeToggle } from '@/components/theme-toggle';
import { Button } from '@/components/ui/button';

// The shell every page shares: a quiet studio header with the wordmark, the
// gallery link, and the theme toggle, plus a minimal footer.
export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="sticky top-0 z-40 border-b border-border/70 bg-background/80 backdrop-blur">
        <div className="mx-auto flex h-16 w-full max-w-6xl items-center gap-3 px-4">
          <Link href="/" className="group flex items-center gap-2.5">
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/15 text-primary transition-colors group-hover:bg-primary/25">
              <Captions className="h-4.5 w-4.5" />
            </span>
            <span className="font-display text-lg font-semibold tracking-tight">
              Subvision
            </span>
          </Link>
          <nav className="ml-auto flex items-center gap-1.5">
            <Button variant="ghost" asChild>
              <Link href="/processes">Gallery</Link>
            </Button>
            <ThemeToggle />
            <Button asChild>
              <Link href="/">New video</Link>
            </Button>
          </nav>
        </div>
      </header>
      <main className="flex-1">{children}</main>
      <footer className="border-t border-border/70">
        <div className="mx-auto flex w-full max-w-6xl items-center justify-between px-4 py-5">
          <p className="text-sm text-muted-foreground">
            Subvision — AI subtitles that match your edit.
          </p>
          <p className="text-xs text-muted-foreground/70">
            whisper.cpp · Remotion · ffmpeg
          </p>
        </div>
      </footer>
    </div>
  );
}
