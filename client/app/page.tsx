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
      <section className="mx-auto w-full max-w-6xl px-4 pb-24 pt-14 md:pt-20">
        <div className="mx-auto max-w-2xl text-center">
          <span className="inline-flex items-center gap-1.5 rounded-full border border-primary/30 bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
            <Sparkles className="h-3.5 w-3.5" />
            AI subtitles, framed and styled your way
          </span>
          <h1 className="mt-5 font-display text-4xl font-bold leading-[1.05] tracking-tight sm:text-5xl md:text-6xl">
              Captions that look
            <span className="text-primary"> directed</span>, not dumped.
          </h1>
          <p className="mx-auto mt-5 max-w-xl text-balance text-lg text-muted-foreground">
            Pick a video, trim it, frame it for any platform, and style your
            captions. Subvision transcribes and burns everything in — you just
            download the result.
          </p>
        </div>

        <div
          role="button"
          tabIndex={0}
          aria-label="Choose a video to edit"
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
          className={`group mx-auto mt-10 flex max-w-2xl cursor-pointer flex-col items-center justify-center rounded-2xl border-2 border-dashed px-6 py-14 text-center transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background sm:py-16 ${
            dragging
              ? 'border-primary bg-primary/10 scale-[1.01]'
              : 'border-border bg-card/50 hover:border-primary/50 hover:bg-card'
          }`}
        >
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
          <span
            className={`flex h-16 w-16 items-center justify-center rounded-2xl border transition-colors ${
              dragging
                ? 'border-primary/50 bg-primary/20 text-primary'
                : 'border-border bg-accent text-muted-foreground group-hover:text-primary'
            }`}
          >
            {dragging ? (
              <Clapperboard className="h-7 w-7" />
            ) : (
              <UploadCloud className="h-7 w-7" />
            )}
          </span>
          <p className="mt-5 font-display text-xl font-semibold">
            {dragging ? 'Drop it — let’s caption it' : 'Drop a video here'}
          </p>
          <p className="mt-1.5 text-sm text-muted-foreground">
            or click to browse — MP4, MOV, AVI, MKV, WebM, up to 500 MB
          </p>
          <Button size="lg" className="mt-6 pointer-events-none">
            <FileVideo2 className="h-4 w-4" />
            Choose a video
          </Button>
        </div>

        <div className="mx-auto mt-14 grid max-w-3xl gap-4 sm:grid-cols-3">
          {[
            {
              icon: Wand2,
              title: 'Frame & trim',
              body: 'Reframe to 9:16, 4:5, 1:1, or anything free. Drag the duration down to the good part.',
            },
            {
              icon: Sparkles,
              title: 'Style the captions',
              body: 'Fonts, outline, highlight swipes — previewed live before a single frame renders.',
            },
            {
              icon: Clapperboard,
              title: 'AI does the rest',
              body: 'whisper.cpp transcribes, the renderer burns styled subtitles in, you download.',
            },
          ].map(({ icon: Icon, title, body }) => (
            <div
              key={title}
              className="rounded-xl border border-border/70 bg-card/40 p-5"
            >
              <Icon className="h-5 w-5 text-primary" />
              <h3 className="mt-3 font-display text-sm font-semibold">{title}</h3>
              <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
                {body}
              </p>
            </div>
          ))}
        </div>
      </section>
    </AppShell>
  );
}
