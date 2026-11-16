'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ArrowDownRight, ArrowRight, Crop, Film, Sparkles, UploadCloud } from 'lucide-react';

import { AppShell } from '@/components/app-shell';
import { CaptionStage } from '@/components/cinematic/caption-stage';
import { Marquee } from '@/components/cinematic/marquee';
import { Reveal } from '@/components/cinematic/reveal';
import { Spotlight } from '@/components/cinematic/spotlight';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import { setPendingVideo } from '@/lib/pending-video';

// The home page does one thing: hand a video to the editor. Picking or
// dropping a file registers it in the in-memory hand-off and routes to
// /editor; nothing uploads from here.

const ACCEPTED_EXTENSIONS = ['.mp4', '.mov', '.avi', '.mkv', '.webm'];

const STEPS = [
  {
    n: '01',
    icon: Crop,
    title: 'Frame it',
    body: 'Reframe footage to 9:16 Shorts, 1:1 square feeds or 16:9 widescreen, then drag to pan and keep the subject centred.',
    tags: ['9:16', '1:1', '16:9', 'Free'],
  },
  {
    n: '02',
    icon: Sparkles,
    title: 'Direct the captions',
    body: 'Four animations (karaoke swipe, pop, slide, fade) with your font, highlight colour, outline and scale. Speech becomes timed words.',
    tags: ['Karaoke', 'Pop', 'Slide', 'Fade'],
  },
  {
    n: '03',
    icon: Film,
    title: 'Render on your server',
    body: 'Captions are burned into the finished MP4 on your own machine, then waiting in the gallery to watch or download.',
    tags: ['MP4', 'Local'],
  },
];

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

  // Keep the drag listeners stable: they call the latest handler through a ref.
  const handleFileRef = useRef(handleFile);
  handleFileRef.current = handleFile;

  // Dropping a file anywhere on the page starts the edit.
  useEffect(() => {
    let depth = 0;
    const hasFiles = (event: DragEvent) =>
      Array.from(event.dataTransfer?.types ?? []).includes('Files');
    const enter = (event: DragEvent) => {
      if (!hasFiles(event)) return;
      depth += 1;
      setDragging(true);
    };
    const leave = (event: DragEvent) => {
      if (!hasFiles(event)) return;
      depth = Math.max(0, depth - 1);
      if (depth === 0) setDragging(false);
    };
    const over = (event: DragEvent) => {
      if (hasFiles(event)) event.preventDefault();
    };
    const drop = (event: DragEvent) => {
      if (!hasFiles(event)) return;
      event.preventDefault();
      depth = 0;
      setDragging(false);
      handleFileRef.current(event.dataTransfer?.files?.[0]);
    };
    window.addEventListener('dragenter', enter);
    window.addEventListener('dragleave', leave);
    window.addEventListener('dragover', over);
    window.addEventListener('drop', drop);
    return () => {
      window.removeEventListener('dragenter', enter);
      window.removeEventListener('dragleave', leave);
      window.removeEventListener('dragover', over);
      window.removeEventListener('drop', drop);
    };
  }, []);

  const choose = () => inputRef.current?.click();

  return (
    <AppShell>
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

      {/* Full-screen drop target */}
      <div
        aria-hidden={!dragging}
        className={`pointer-events-none fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-xl transition-opacity duration-300 ${
          dragging ? 'opacity-100' : 'opacity-0'
        }`}
      >
        <div className="m-6 flex h-[calc(100%-3rem)] w-[calc(100%-3rem)] flex-col items-center justify-center gap-4 rounded-[2.5rem] border border-dashed border-primary/70 bg-primary/[0.06]">
          <UploadCloud className="h-12 w-12 text-primary" />
          <p className="text-4xl font-semibold tracking-[-0.04em] sm:text-6xl">Drop to begin</p>
        </div>
      </div>

      {/* Hero */}
      <section className="relative mx-auto w-full max-w-6xl px-4 pb-10 pt-10 sm:px-6 md:pt-16">
        <div className="rise inline-flex items-center gap-2 rounded-full border border-border bg-foreground/[0.04] py-1 pl-1.5 pr-3.5 text-xs font-medium tracking-tight text-muted-foreground backdrop-blur">
          <span className="flex items-center gap-1.5 rounded-full bg-primary/15 px-2 py-0.5 text-primary">
            <span className="h-1.5 w-1.5 rounded-full bg-primary motion-safe:animate-[blink_1.6s_ease-in-out_infinite]" />
            Local-first
          </span>
          <span className="truncate">Runs on your own server</span>
        </div>

        <h1 className="mt-7 text-[clamp(3.4rem,12.5vw,10.5rem)] font-semibold leading-[0.88] tracking-[-0.055em]">
          <span className="line-mask">
            <span style={{ '--line-delay': '60ms' } as React.CSSProperties}>Captions,</span>
          </span>
          <span className="line-mask">
            <span
              className="text-gradient"
              style={{ '--line-delay': '200ms' } as React.CSSProperties}
            >
              directed.
            </span>
          </span>
        </h1>

        <div className="mt-8 flex flex-col justify-between gap-8 md:flex-row md:items-end">
          <p className="rise rise-3 max-w-xl text-balance text-lg leading-relaxed text-muted-foreground sm:text-xl">
            Reframe footage for Reels, Shorts and TikTok. Pick an animated caption
            style, then render the MP4 on your own machine.
          </p>
          <div className="rise rise-4 flex flex-col gap-3 min-[480px]:flex-row min-[480px]:items-center">
            <Button size="lg" onClick={choose} className="group gap-2.5 max-[479px]:w-full">
              <UploadCloud className="h-4.5 w-4.5" />
              Drop a video
              <ArrowRight className="h-4 w-4 transition-transform duration-200 group-hover:translate-x-0.5" />
            </Button>
            <Button size="lg" variant="outline" asChild className="max-[479px]:w-full">
              <Link href="/processes">Open gallery</Link>
            </Button>
          </div>
        </div>

        <div className="rise rise-4 mt-12 sm:mt-16">
          <CaptionStage />
        </div>
      </section>

      {/* Ticker */}
      <section aria-label="What it does" className="mt-14 border-y border-border py-6 text-4xl sm:text-6xl">
        <Marquee items={['Trim', 'Reframe', 'Transcribe', 'Animate', 'Export']} />
      </section>

      {/* Dropzone */}
      <section className="mx-auto mt-24 w-full max-w-6xl px-4 sm:px-6">
        <Reveal>
          <Spotlight className="group relative overflow-hidden rounded-[2rem] border border-border bg-card/60 p-1.5 shadow-soft backdrop-blur-sm">
            <div
              role="button"
              tabIndex={0}
              aria-label="Choose a video file to edit"
              onClick={choose}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault();
                  choose();
                }
              }}
              className="relative flex cursor-pointer flex-col items-center justify-center gap-6 rounded-[1.6rem] border border-dashed border-foreground/20 px-6 py-16 text-center transition-colors duration-300 hover:border-primary/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:py-24"
            >
              <div className="relative flex h-24 w-24 items-center justify-center">
                <span
                  aria-hidden
                  className="absolute inset-0 rounded-full border border-dashed border-primary/60 motion-safe:animate-[spin-slow_18s_linear_infinite]"
                />
                <span className="flex h-16 w-16 items-center justify-center rounded-full bg-brand text-primary-foreground shadow-[0_14px_40px_-10px_rgb(124_108_255/0.9)] transition-transform duration-300 group-hover:scale-110">
                  <UploadCloud className="h-7 w-7" />
                </span>
              </div>
              <div>
                <h2 className="text-3xl font-semibold tracking-[-0.04em] sm:text-5xl">
                  Start with a video
                </h2>
                <p className="mx-auto mt-3 max-w-md text-muted-foreground">
                  Drop a file anywhere on this page, or click to browse. Nothing uploads until you
                  press generate in the editor.
                </p>
              </div>
              <div className="flex flex-wrap items-center justify-center gap-2 font-mono text-xs text-muted-foreground">
                {['.mp4', '.mov', '.webm', '.mkv'].map((ext) => (
                  <span key={ext} className="rounded-full border border-border bg-foreground/[0.04] px-3 py-1">
                    {ext}
                  </span>
                ))}
                <span className="px-1">up to 500 MB</span>
              </div>
            </div>
          </Spotlight>
        </Reveal>
      </section>

      {/* Steps */}
      <section className="mx-auto mt-28 w-full max-w-6xl px-4 sm:px-6">
        <Reveal className="mb-10 flex items-end justify-between gap-6">
          <h2 className="text-4xl font-semibold leading-[0.95] tracking-[-0.045em] sm:text-6xl">
            Three decisions.
            <br />
            <span className="text-muted-foreground">One render.</span>
          </h2>
          <ArrowDownRight className="hidden h-12 w-12 text-primary sm:block" />
        </Reveal>

        <div className="grid gap-4 md:grid-cols-3">
          {STEPS.map((step, index) => (
            <Reveal key={step.n} delay={index * 110}>
              <Spotlight className="flex h-full flex-col rounded-3xl border border-border bg-card/60 p-7 shadow-soft backdrop-blur-sm transition-transform duration-500 hover:-translate-y-1">
                <div className="flex items-start justify-between">
                  <span className="flex h-11 w-11 items-center justify-center rounded-2xl bg-primary/15 text-primary">
                    <step.icon className="h-5 w-5" />
                  </span>
                  <span className="font-mono text-sm text-muted-foreground">{step.n}</span>
                </div>
                <h3 className="mt-10 text-2xl font-semibold tracking-[-0.03em]">{step.title}</h3>
                <p className="mt-3 flex-1 text-sm leading-relaxed text-muted-foreground">
                  {step.body}
                </p>
                <div className="mt-6 flex flex-wrap gap-1.5">
                  {step.tags.map((tag) => (
                    <span
                      key={tag}
                      className="rounded-full border border-border bg-foreground/[0.04] px-2.5 py-1 font-mono text-[11px] text-muted-foreground"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              </Spotlight>
            </Reveal>
          ))}
        </div>
      </section>
    </AppShell>
  );
}
