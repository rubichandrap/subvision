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
        {/* Hero */}
        <div className="mx-auto max-w-3xl text-center">
          <h1 className="font-display text-4xl font-extrabold leading-[1.05] tracking-tight sm:text-5xl md:text-6xl text-balance">
            Captions that look{' '}
            <span className="relative inline-block border-2 border-border bg-primary px-3 py-0.5 text-primary-foreground shadow-brutal">
              directed
            </span>
            , not dumped.
          </h1>

          <p className="mx-auto mt-5 max-w-2xl text-balance text-base leading-relaxed text-muted-foreground sm:text-lg">
            Reframe footage for TikTok, Reels, or YouTube Shorts. Choose an animated caption
            style, then render the MP4 on your server.
          </p>
        </div>
        {/* Dropzone */}
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
            className={`group relative flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-border bg-card p-8 shadow-brutal transition-all duration-100 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring focus-visible:ring-offset-0 sm:p-12 text-center ${
              dragging
                ? 'bg-primary -translate-x-0.5 -translate-y-0.5 shadow-brutal-lg'
                : 'hover:-translate-x-0.5 hover:-translate-y-0.5 hover:bg-accent/30 hover:shadow-brutal-lg'
            }`}
          >
            {/* Corner brackets */}
            <div className="pointer-events-none absolute left-3 top-3 h-3 w-3 border-l-[3px] border-t-[3px] border-border" />
            <div className="pointer-events-none absolute right-3 top-3 h-3 w-3 border-r-[3px] border-t-[3px] border-border" />
            <div className="pointer-events-none absolute bottom-3 left-3 h-3 w-3 border-l-[3px] border-b-[3px] border-border" />
            <div className="pointer-events-none absolute bottom-3 right-3 h-3 w-3 border-r-[3px] border-b-[3px] border-border" />

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
              className={`flex h-16 w-16 items-center justify-center border-2 border-border shadow-brutal-sm transition-all duration-100 ${
                dragging
                  ? 'bg-secondary text-secondary-foreground scale-110'
                  : 'bg-primary text-primary-foreground group-hover:-translate-y-1 group-hover:bg-secondary group-hover:text-secondary-foreground'
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
                {dragging ? 'Drop to load your video' : 'Drop your video here'}
              </p>
              <p className="mt-1.5 text-xs sm:text-sm text-muted-foreground">
                or click anywhere to browse local files
              </p>
            </div>

            {/* Supported format badges */}
            <div className="mt-4 flex flex-wrap items-center justify-center gap-1.5 font-mono text-[11px] font-bold text-muted-foreground">
              {['MP4', 'MOV', 'WEBM', 'MKV'].map((ext) => (
                <span
                  key={ext}
                  className="border-2 border-border bg-muted px-2 py-0.5"
                >
                  .{ext.toLowerCase()}
                </span>
              ))}
              <span className="text-muted-foreground/60">· up to 500 MB</span>
            </div>

            <Button
              size="lg"
              className="mt-6 pointer-events-none gap-2 font-bold transition-transform duration-100 group-hover:-translate-x-0.5 group-hover:-translate-y-0.5"
            >
              <FileVideo2 className="h-4 w-4" />
              Select video file
            </Button>
          </div>
        </div>

        {/* Pipeline Overview */}
        <div className="mx-auto mt-16 max-w-4xl">
          <div className="mb-4 flex items-center justify-between border-b-2 border-border pb-3">
            <div className="flex items-center gap-2">
              <span className="h-2.5 w-2.5 border-2 border-border bg-primary" />
              <h2 className="font-mono text-xs font-bold uppercase tracking-widest text-muted-foreground">
                Pipeline
              </h2>
            </div>
            <span className="font-mono text-xs font-bold uppercase tracking-wide text-muted-foreground">
              Trim · Reframe · Transcribe · Export
            </span>
          </div>

          <div className="grid gap-5 sm:grid-cols-3">
            <div className="border-2 border-border bg-card p-5 shadow-brutal transition-all duration-100 hover:-translate-x-0.5 hover:-translate-y-0.5 hover:shadow-brutal-lg">
              <div className="flex h-9 w-9 items-center justify-center border-2 border-border bg-accent text-accent-foreground">
                <Wand2 className="h-4.5 w-4.5" />
              </div>
              <h3 className="mt-3.5 font-display text-sm font-bold">
                Reframe and crop
              </h3>
              <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                Reframe footage to 9:16 Shorts, 1:1 square feeds, or 16:9 widescreen. Drag to pan
                and center the subject.
              </p>
              <div className="mt-3 flex gap-1.5 font-mono text-[10px] font-bold text-muted-foreground">
                <span className="border border-border bg-muted px-1.5 py-0.5">9:16</span>
                <span className="border border-border bg-muted px-1.5 py-0.5">1:1</span>
                <span className="border border-border bg-muted px-1.5 py-0.5">16:9</span>
              </div>
            </div>

            <div className="border-2 border-border bg-card p-5 shadow-brutal transition-all duration-100 hover:-translate-x-0.5 hover:-translate-y-0.5 hover:shadow-brutal-lg">
              <div className="flex h-9 w-9 items-center justify-center border-2 border-border bg-secondary text-secondary-foreground">
                <Sparkles className="h-4.5 w-4.5" />
              </div>
              <h3 className="mt-3.5 font-display text-sm font-bold">
                Animated captions
              </h3>
              <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                Choose from four styles: karaoke swipe, pop, slide, or fade. Configure font
                family, highlight color, outline stroke, and scale.
              </p>
              <div className="mt-3 flex gap-1.5 font-mono text-[10px] font-bold text-muted-foreground">
                <span className="border border-border bg-muted px-1.5 py-0.5">Karaoke</span>
                <span className="border border-border bg-muted px-1.5 py-0.5">Pop</span>
                <span className="border border-border bg-muted px-1.5 py-0.5">Slide</span>
              </div>
            </div>

            <div className="border-2 border-border bg-card p-5 shadow-brutal transition-all duration-100 hover:-translate-x-0.5 hover:-translate-y-0.5 hover:shadow-brutal-lg">
              <div className="flex h-9 w-9 items-center justify-center border-2 border-border bg-primary text-primary-foreground">
                <Clapperboard className="h-4.5 w-4.5" />
              </div>
              <h3 className="mt-3.5 font-display text-sm font-bold">
                Local render
              </h3>
              <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                Speech becomes timed captions, burned into the finished MP4 on your server.
              </p>
            </div>
          </div>
        </div>
      </section>
    </AppShell>
  );
}
