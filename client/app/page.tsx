'use client';

import { useCallback, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Clapperboard, FileVideo2, Sparkles, UploadCloud, Wand2 } from 'lucide-react';

import { AppShell } from '@/components/app-shell';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import { setPendingVideo } from '@/lib/pending-video';

// The home page does one thing: hand a video to the editor. Picking or
// dropping a file registers it in the in-memory hand-off and routes to
// /editor; nothing uploads from here.

const ACCEPTED_EXTENSIONS = ['.mp4', '.mov', '.avi', '.mkv', '.webm'];

export default function Home() {
  const router = useRouter();
  const { toast } = useToast();
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);

  const handleFile = useCallback(
    (file: File | undefined) => {
      if (!file) return;
      const extension = `.${file.name.split('.').pop()?.toLowerCase() ?? ''}`;
      const looksLikeVideo =
        file.type.startsWith('video/') ||
        ACCEPTED_EXTENSIONS.includes(extension);
      if (!looksLikeVideo) {
        toast({
          title: 'That is not a video file',
          description: `Subvision accepts ${ACCEPTED_EXTENSIONS.join(', ')}.`,
          variant: 'destructive',
        });
        return;
      }
      const id = setPendingVideo(file);
      router.push(`/editor?file=${encodeURIComponent(id)}`);
    },
    [router, toast]
  );

  return (
    <AppShell>
      <section className="mx-auto w-full max-w-6xl px-4 pb-24 pt-10 sm:px-6 md:pt-16">
        {/* Studio Hero Header */}
        <div className="mx-auto max-w-3xl text-center">
          <div className="inline-flex items-center gap-2 rounded-full border border-border/80 bg-card/80 px-3.5 py-1 text-xs font-medium text-muted-foreground shadow-2xs backdrop-blur-sm">
            <span className="flex h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
            <span className="font-semibold text-foreground">Subvision Studio</span>
            <span className="text-border">/</span>
            <span>whisper.cpp</span>
            <span className="text-border">/</span>
            <span className="hidden sm:inline">Remotion</span>
          </div>

          <h1 className="mt-6 font-display text-4xl font-bold leading-[1.1] tracking-tight sm:text-5xl md:text-6xl text-balance">
            Captions that look{' '}
            <span className="relative inline-block">
              <span className="relative z-10 text-primary underline decoration-primary/40 decoration-wavy underline-offset-4">
                directed
              </span>
            </span>
            , not dumped.
          </h1>

          <p className="mx-auto mt-5 max-w-2xl text-balance text-base leading-relaxed text-muted-foreground sm:text-lg">
            Reframe footage for TikTok, Reels, or YouTube Shorts. Choose a kinetic subtitle
            animation, then render the MP4 on your server.
          </p>
        </div>
        {/* Studio Ingestion Deck */}
        <div className="mx-auto mt-10 max-w-2xl">
          <div
            role="button"
            tabIndex={0}
            aria-label="Choose a video file to edit"
            onClick={() => inputRef.current?.click()}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                inputRef.current?.click();
              }
            }}
            onDragOver={(event) => {
              event.preventDefault();
              setDragging(true);
            }}
            onDragLeave={() => setDragging(false)}
            onDrop={(event) => {
              event.preventDefault();
              setDragging(false);
              handleFile(event.dataTransfer.files?.[0]);
            }}
            className={`group relative flex cursor-pointer flex-col items-center justify-center rounded-2xl border transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background p-8 sm:p-12 text-center ${
              dragging
                ? 'border-primary bg-primary/10 shadow-lg scale-[1.01]'
                : 'border-border/80 bg-card/60 hover:border-primary/60 hover:bg-card hover:shadow-md'
            }`}
          >
            {/* Studio framing viewfinder brackets */}
            <div className="pointer-events-none absolute left-3 top-3 h-3 w-3 border-l-2 border-t-2 border-primary/40 transition-colors group-hover:border-primary" />
            <div className="pointer-events-none absolute right-3 top-3 h-3 w-3 border-r-2 border-t-2 border-primary/40 transition-colors group-hover:border-primary" />
            <div className="pointer-events-none absolute bottom-3 left-3 h-3 w-3 border-l-2 border-b-2 border-primary/40 transition-colors group-hover:border-primary" />
            <div className="pointer-events-none absolute bottom-3 right-3 h-3 w-3 border-r-2 border-b-2 border-primary/40 transition-colors group-hover:border-primary" />

            <input
              ref={inputRef}
              type="file"
              accept="video/*"
              className="hidden"
              onChange={(event) => {
                handleFile(event.target.files?.[0]);
                event.target.value = '';
              }}
            />

            <div
              className={`flex h-16 w-16 items-center justify-center rounded-2xl border transition-all duration-200 ${
                dragging
                  ? 'border-primary bg-primary/20 text-primary scale-110'
                  : 'border-border/80 bg-accent/60 text-muted-foreground group-hover:border-primary/50 group-hover:bg-primary/10 group-hover:text-primary'
              }`}
            >
              {dragging ? (
                <Clapperboard className="h-8 w-8 animate-bounce" />
              ) : (
                <UploadCloud className="h-8 w-8 transition-transform duration-200 group-hover:-translate-y-0.5" />
              )}
            </div>

            <div className="mt-5">
              <p className="font-display text-xl font-semibold tracking-tight">
                {dragging ? 'Drop to load footage into studio' : 'Drop video footage here'}
              </p>
              <p className="mt-1.5 text-xs sm:text-sm text-muted-foreground">
                or click anywhere to browse local files
              </p>
            </div>

            {/* Supported format badges */}
            <div className="mt-4 flex flex-wrap items-center justify-center gap-1.5 font-mono text-[11px] text-muted-foreground">
              {['MP4', 'MOV', 'WEBM', 'MKV'].map((ext) => (
                <span
                  key={ext}
                  className="rounded-md border border-border/80 bg-muted/50 px-2 py-0.5 font-medium"
                >
                  .{ext.toLowerCase()}
                </span>
              ))}
              <span className="text-muted-foreground/60">· up to 500 MB</span>
            </div>

            <Button
              size="lg"
              className="mt-6 pointer-events-none gap-2 font-medium shadow-sm transition-transform duration-200 group-hover:scale-105"
            >
              <FileVideo2 className="h-4 w-4" />
              Select video file
            </Button>
          </div>
        </div>

        {/* Pipeline Overview */}
        <div className="mx-auto mt-16 max-w-4xl">
          <div className="mb-4 flex items-center justify-between border-b border-border/60 pb-3">
            <div className="flex items-center gap-2">
              <span className="h-1.5 w-1.5 rounded-full bg-primary" />
              <h2 className="font-display text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Pipeline
              </h2>
            </div>
            <span className="text-xs text-muted-foreground">
              Trim · Reframe · Transcribe · Burn
            </span>
          </div>

          <div className="grid gap-4 sm:grid-cols-3">
            <div className="rounded-xl border border-border/80 bg-card/60 p-5 transition-colors hover:border-border">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-accent/60 text-primary">
                <Wand2 className="h-4.5 w-4.5" />
              </div>
              <h3 className="mt-3.5 font-display text-sm font-semibold">
                Reframe and crop
              </h3>
              <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                Reframe footage to 9:16 Shorts, 1:1 square feeds, or 16:9 widescreen. Drag to pan
                and center the subject.
              </p>
              <div className="mt-3 flex gap-1.5 font-mono text-[10px] text-muted-foreground">
                <span className="rounded bg-muted px-1.5 py-0.5">9:16</span>
                <span className="rounded bg-muted px-1.5 py-0.5">1:1</span>
                <span className="rounded bg-muted px-1.5 py-0.5">16:9</span>
              </div>
            </div>

            <div className="rounded-xl border border-border/80 bg-card/60 p-5 transition-colors hover:border-border">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-accent/60 text-primary">
                <Sparkles className="h-4.5 w-4.5" />
              </div>
              <h3 className="mt-3.5 font-display text-sm font-semibold">
                Animated captions
              </h3>
              <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                Choose from four styles: karaoke swipe, pop, slide, or fade. Configure font
                family, highlight color, outline stroke, and scale.
              </p>
              <div className="mt-3 flex gap-1.5 font-mono text-[10px] text-muted-foreground">
                <span className="rounded bg-muted px-1.5 py-0.5">Karaoke</span>
                <span className="rounded bg-muted px-1.5 py-0.5">Pop</span>
                <span className="rounded bg-muted px-1.5 py-0.5">Slide</span>
              </div>
            </div>

            <div className="rounded-xl border border-border/80 bg-card/60 p-5 transition-colors hover:border-border">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-accent/60 text-primary">
                <Clapperboard className="h-4.5 w-4.5" />
              </div>
              <h3 className="mt-3.5 font-display text-sm font-semibold">
                Local render
              </h3>
              <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                whisper.cpp transcribes speech to timestamped tokens. Remotion composites styled
                captions directly into the MP4 file.
              </p>
              <div className="mt-3 flex gap-1.5 font-mono text-[10px] text-muted-foreground">
                <span className="rounded bg-muted px-1.5 py-0.5">whisper.cpp</span>
                <span className="rounded bg-muted px-1.5 py-0.5">Remotion</span>
              </div>
            </div>
          </div>
        </div>
      </section>
    </AppShell>
  );
}
