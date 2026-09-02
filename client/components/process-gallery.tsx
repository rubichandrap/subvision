'use client';

import Link from 'next/link';
import { formatDistanceToNow } from 'date-fns';
import { Clapperboard, FileVideo2, Loader2 } from 'lucide-react';

import { StageBadge } from '@/components/process-stage';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { useProcessList } from '@/hooks/use-process-polling';
import { downloadUrl, type Process } from '@/lib/api';

// The gallery: every Process as a card. Finished videos preview on hover
// straight from the server's download endpoint — no thumbnails, no server
// changes; in-flight jobs show their live stage instead.

function GalleryCard({ process }: { process: Process }) {
  const done = process.stage === 'done' && Boolean(process.downloadUrl);
  const src = done ? downloadUrl(process) : null;

  return (
    <Link href={`/processes/${process.id}`} className="group block">
      <Card className="overflow-hidden border-border/70 bg-card/50 p-0 transition-colors group-hover:border-primary/40">
        <div className="relative aspect-video bg-black/50">
          {src ? (
            <video
              src={`${src}#t=0.5`}
              muted
              loop
              playsInline
              preload="metadata"
              className="h-full w-full object-contain"
              onMouseEnter={(event) => {
                void event.currentTarget.play().catch(() => {});
              }}
              onMouseLeave={(event) => {
                event.currentTarget.pause();
                event.currentTarget.currentTime = 0.5;
              }}
            />
          ) : (
            <div className="flex h-full w-full flex-col items-center justify-center gap-3">
              <FileVideo2 className="h-10 w-10 text-muted-foreground/50" />
              {process.stage === 'failed' ? (
                <p className="max-w-[85%] truncate text-xs text-red-400">
                  {process.reason || 'Render failed'}
                </p>
              ) : (
                <p className="text-xs text-muted-foreground">Rendering your video…</p>
              )}
            </div>
          )}
          {/* A dark plate keeps the badge readable over bright video frames */}
          <div className="absolute right-2 top-2 rounded-full bg-black/55 p-0.5 backdrop-blur-sm">
            <StageBadge stage={process.stage} />
          </div>
        </div>
        <div className="p-3">
          <p className="truncate text-sm font-medium">{process.filename || process.id}</p>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {formatDistanceToNow(new Date(process.createdAt), { addSuffix: true })}
          </p>
        </div>
      </Card>
    </Link>
  );
}

export function ProcessGallery() {
  const { processes, error } = useProcessList();

  if (processes === null && !error) {
    return (
      <div className="flex justify-center items-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error && processes === null) {
    return (
      <Card className="flex flex-col items-center justify-center border-border/70 bg-card/50 py-16">
        <FileVideo2 className="mb-4 h-10 w-10 text-muted-foreground/60" />
        <h3 className="font-display text-lg font-semibold">Could not load your gallery</h3>
        <p className="mt-2 text-center text-sm text-muted-foreground">{error}</p>
      </Card>
    );
  }

  if (processes !== null && processes.length === 0) {
    return (
      <Card className="flex flex-col items-center justify-center border-border/70 bg-card/50 py-16">
        <Clapperboard className="mb-4 h-10 w-10 text-muted-foreground/60" />
        <h3 className="font-display text-lg font-semibold">Nothing here yet</h3>
        <p className="mt-2 text-center text-sm text-muted-foreground">
          Caption your first video and it will show up here.
        </p>
        <Button asChild className="mt-6">
          <Link href="/">Choose a video</Link>
        </Button>
      </Card>
    );
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {(processes ?? []).map((process) => (
        <GalleryCard key={process.id} process={process} />
      ))}
    </div>
  );
}
