'use client';

import * as React from 'react';
import Link from 'next/link';
import { formatDistanceToNow } from 'date-fns';
import {
  AlertTriangle,
  Download,
  FileVideo2,
  Film,
  Loader2,
  Play,
  Plus,
  Trash2,
} from 'lucide-react';

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { StageBadge } from '@/components/process-stage';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { useProcessDelete, useProcessList } from '@/hooks/use-process-polling';
import { IN_FLIGHT_STAGES, downloadUrl, type Process, type ProcessStage } from '@/lib/api';
import { cn } from '@/lib/utils';

// The gallery: every Process as a card. Finished videos preview on hover
// straight from the server's download endpoint — no thumbnails, no server
// changes; in-flight jobs show their live stage instead. Delete lives on the
// card too (ADR-0004): an immediate, irreversible erase of the record and
// both stored objects.

export function DeleteProcessDialog({
  process,
  open,
  onOpenChange,
  onDeleted,
  trigger,
}: {
  process: Pick<Process, 'id' | 'filename'>;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
  trigger?: React.ReactNode;
}) {
  const { removeProcess, deleting } = useProcessDelete();
  const [error, setError] = React.useState<string | null>(null);

  const handleDelete = async () => {
    setError(null);
    try {
      await removeProcess(process.id);
      onOpenChange(false);
      onDeleted?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete the process');
    }
  };

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      {trigger}
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete this process?</AlertDialogTitle>
          <AlertDialogDescription>
            “{process.filename || process.id}” and its rendered result will be permanently
            removed from the gallery and storage. This cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        {error && <p className="text-sm text-red-400">{error}</p>}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            disabled={deleting}
            onClick={(event) => {
              event.preventDefault();
              void handleDelete();
            }}
            className="bg-red-600 text-white hover:bg-red-600/90"
          >
            {deleting && <Loader2 className="h-4 w-4 animate-spin" />}
            Delete forever
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

export function DeleteProcessButton({
  process,
  onDeleted,
  className,
  label = 'Delete',
}: {
  process: Pick<Process, 'id' | 'filename'>;
  onDeleted?: () => void;
  className?: string;
  label?: string;
}) {
  const [open, setOpen] = React.useState(false);

  return (
    <DeleteProcessDialog
      process={process}
      open={open}
      onOpenChange={setOpen}
      onDeleted={onDeleted}
      trigger={
        <button
          type="button"
          aria-label={label}
          className={className}
          onClick={(event) => {
            event.preventDefault();
            event.stopPropagation();
            setOpen(true);
          }}
        >
          <Trash2 className="h-4 w-4" />
        </button>
      }
    />
  );
}

function GalleryCard({ process }: { process: Process }) {
  const done = process.stage === 'done' && Boolean(process.downloadUrl);
  const src = done ? downloadUrl(process) : null;
  const inFlight = IN_FLIGHT_STAGES.has(process.stage);
  const failed = process.stage === 'failed';

  return (
    <div className="group relative transition-all duration-200 hover:-translate-y-0.5">
      <Link href={`/processes/${process.id}`} className="block focus-visible:outline-none">
        <Card className="overflow-hidden rounded-xl border border-border/70 bg-card p-0 shadow-xs transition-all duration-200 group-hover:border-primary/50 group-hover:shadow-md">
          {/* Video / Preview Stage */}
          <div className="relative aspect-video overflow-hidden bg-neutral-900 dark:bg-neutral-950">
            {src ? (
              <>
                {/* Ambient blurred backdrop fills pillarbox for vertical / 9:16 videos */}
                <video
                  src={`${src}#t=0.5`}
                  muted
                  playsInline
                  preload="metadata"
                  aria-hidden="true"
                  className="pointer-events-none absolute inset-0 h-full w-full object-cover opacity-35 blur-xl scale-125"
                />
                {/* Subtle vignette */}
                <div className="pointer-events-none absolute inset-0 bg-black/20" />

                {/* Primary video element */}
                <video
                  src={`${src}#t=0.5`}
                  muted
                  loop
                  playsInline
                  preload="metadata"
                  className="relative z-10 h-full w-full object-contain"
                  onMouseEnter={(event) => {
                    void event.currentTarget.play().catch(() => {});
                  }}
                  onMouseLeave={(event) => {
                    event.currentTarget.pause();
                    event.currentTarget.currentTime = 0.5;
                  }}
                />

                {/* Hover Play Hint Overlay */}
                <div className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center bg-black/25 opacity-0 backdrop-blur-[1px] transition-opacity duration-200 group-hover:opacity-100">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-white/90 text-black shadow-lg">
                    <Play className="h-4 w-4 fill-current ml-0.5" />
                  </div>
                </div>

                {/* Direct Download Button */}
                <a
                  href={src}
                  download={process.filename || 'captioned-video.mp4'}
                  onClick={(e) => e.stopPropagation()}
                  className="absolute bottom-2.5 left-2.5 z-30 inline-flex h-7 items-center gap-1.5 rounded-md border border-border/80 bg-background/90 px-2 text-xs font-medium text-foreground opacity-0 shadow-xs backdrop-blur-md transition-all duration-200 hover:bg-background group-hover:opacity-100"
                  title="Download MP4"
                >
                  <Download className="h-3.5 w-3.5" />
                  <span>Download</span>
                </a>
              </>
            ) : failed ? (
              <div className="flex h-full w-full flex-col items-center justify-center gap-2 p-4 text-center bg-rose-500/5 dark:bg-rose-950/20">
                <div className="flex h-9 w-9 items-center justify-center rounded-full border border-rose-500/30 bg-rose-500/10 text-rose-500">
                  <AlertTriangle className="h-4 w-4" />
                </div>
                <div>
                  <p className="text-xs font-semibold text-rose-600 dark:text-rose-400">
                    Render Interrupted
                  </p>
                  <p className="mt-1 max-w-[240px] truncate rounded border border-rose-500/20 bg-rose-500/10 px-2 py-0.5 font-mono text-[10px] text-rose-700 dark:text-rose-300">
                    {process.reason || 'Process failed during execution'}
                  </p>
                </div>
              </div>
            ) : (
              <div className="flex h-full w-full flex-col items-center justify-center gap-2.5 p-4 text-center bg-accent/20">
                <div className="relative flex h-10 w-10 items-center justify-center rounded-full border border-primary/30 bg-primary/10 text-primary">
                  <Loader2 className="h-5 w-5 animate-spin" />
                </div>
                <div>
                  <p className="text-xs font-medium text-foreground">
                    {process.stage === 'transcribing'
                      ? 'Transcribing audio…'
                      : 'Rendering captions…'}
                  </p>
                  <p className="mt-0.5 text-[10px] text-muted-foreground">
                    whisper.cpp processing
                  </p>
                </div>
              </div>
            )}

            {/* Stage Badge on Top Right */}
            <div className="absolute right-2.5 top-2.5 z-30">
              <StageBadge stage={process.stage} />
            </div>
          </div>

          {/* Card Meta Content */}
          <div className="flex items-center justify-between p-3.5 pr-11">
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold tracking-tight text-foreground">
                {process.filename || process.id}
              </p>
              <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
                <span>
                  {formatDistanceToNow(new Date(process.createdAt), { addSuffix: true })}
                </span>
                <span>·</span>
                <span className="font-mono text-[10px] text-muted-foreground/80 uppercase">
                  MP4
                </span>
              </div>
            </div>
          </div>
        </Card>
      </Link>

      {/* Delete Process Button */}
      <DeleteProcessButton
        process={process}
        className="absolute bottom-3 right-3 z-30 inline-flex h-8 w-8 items-center justify-center rounded-md border border-transparent text-muted-foreground/60 opacity-0 transition-all duration-200 hover:border-red-500/20 hover:bg-red-500/10 hover:text-red-500 focus-visible:opacity-100 group-hover:opacity-100"
        label={`Delete ${process.filename || process.id}`}
      />
    </div>
  );
}

export function ProcessGallery() {
  const { processes, error } = useProcessList();
  const [filter, setFilter] = React.useState<'all' | 'done' | 'inflight' | 'failed'>('all');

  if (processes === null && !error) {
    return (
      <div className="flex flex-col items-center justify-center py-28 text-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
        <p className="mt-3 text-sm text-muted-foreground">Loading studio gallery…</p>
      </div>
    );
  }

  if (error && processes === null) {
    return (
      <Card className="flex flex-col items-center justify-center rounded-2xl border-border/80 bg-card/60 py-16 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-xl border border-destructive/30 bg-destructive/10 text-destructive">
          <AlertTriangle className="h-6 w-6" />
        </div>
        <h3 className="mt-4 font-display text-base font-semibold">Could not load gallery</h3>
        <p className="mt-1 max-w-sm text-xs text-muted-foreground">{error}</p>
      </Card>
    );
  }

  if (processes !== null && processes.length === 0) {
    return (
      <Card className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border/80 bg-card/40 py-20 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-2xl border border-border/80 bg-accent/60 text-muted-foreground">
          <Film className="h-7 w-7" />
        </div>
        <h3 className="mt-4 font-display text-lg font-semibold">No captioned videos yet</h3>
        <p className="mt-1.5 max-w-sm text-xs leading-relaxed text-muted-foreground">
          Import a video in the studio to reframe it and burn styled subtitles.
        </p>
        <Button asChild className="mt-6 gap-2">
          <Link href="/">
            <Plus className="h-4 w-4" />
            Start a new video
          </Link>
        </Button>
      </Card>
    );
  }

  const allProcesses = processes ?? [];
  const doneCount = allProcesses.filter((p) => p.stage === 'done').length;
  const inflightCount = allProcesses.filter((p) => IN_FLIGHT_STAGES.has(p.stage)).length;
  const failedCount = allProcesses.filter((p) => p.stage === 'failed').length;

  const filteredProcesses = allProcesses.filter((process) => {
    if (filter === 'done') return process.stage === 'done';
    if (filter === 'inflight') return IN_FLIGHT_STAGES.has(process.stage);
    if (filter === 'failed') return process.stage === 'failed';
    return true;
  });

  return (
    <div className="space-y-6">
      {/* Filter tabs */}
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border/60 pb-3">
        <div className="flex items-center gap-1.5">
          {[
            { id: 'all', label: 'All', count: allProcesses.length },
            { id: 'done', label: 'Completed', count: doneCount },
            { id: 'inflight', label: 'In Progress', count: inflightCount },
            { id: 'failed', label: 'Failed', count: failedCount },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setFilter(tab.id as typeof filter)}
              className={cn(
                'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
                filter === tab.id
                  ? 'bg-accent font-semibold text-foreground'
                  : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground'
              )}
            >
              <span>{tab.label}</span>
              <span
                className={cn(
                  'rounded-full px-1.5 py-0.2 text-[10px]',
                  filter === tab.id
                    ? 'bg-primary/20 text-primary font-bold'
                    : 'bg-muted text-muted-foreground'
                )}
              >
                {tab.count}
              </span>
            </button>
          ))}
        </div>

        <p className="text-xs text-muted-foreground">
          Hover to preview · Click for full timeline & details
        </p>
      </div>

      {filteredProcesses.length === 0 ? (
        <div className="py-14 text-center text-xs text-muted-foreground">
          No videos match the selected &quot;{filter}&quot; filter.
        </div>
      ) : (
        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {filteredProcesses.map((process) => (
            <GalleryCard key={process.id} process={process} />
          ))}
        </div>
      )}
    </div>
  );
}
