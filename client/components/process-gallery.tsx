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
        {error && <p className="rounded-xl bg-destructive/15 p-3 text-sm font-medium text-destructive">{error}</p>}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            disabled={deleting}
            onClick={(event) => {
              event.preventDefault();
              void handleDelete();
            }}
            className="bg-destructive text-destructive-foreground shadow-none hover:bg-destructive hover:brightness-110"
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
  const downloadSrc = done ? downloadUrl(process, { download: true }) : undefined;
  const inFlight = IN_FLIGHT_STAGES.has(process.stage);
  const failed = process.stage === 'failed';

  return (
    <div className="group relative min-w-0 transition-transform duration-500 hover:-translate-y-1">
      <Link href={`/processes/${process.id}`} className="block focus-visible:outline-none">
        <Card className="overflow-hidden rounded-3xl p-0 transition-shadow duration-500 group-hover:shadow-soft-lg">
          {/* Video / Preview Stage — stays neutral black */}
          <div className="relative aspect-video overflow-hidden border-b border-border bg-neutral-950">
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
                <div className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center bg-black/25 opacity-0 backdrop-blur-[2px] transition-opacity duration-300 group-hover:opacity-100">
                  <div className="flex h-12 w-12 items-center justify-center rounded-full bg-white text-black shadow-soft-lg">
                    <Play className="h-4 w-4 fill-current ml-0.5" />
                  </div>
                </div>

                {/* Direct Download Button. A button, not an anchor: the card is
                    already a link, and anchors cannot nest. */}
                <button
                  type="button"
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    const link = document.createElement('a');
                    link.href = downloadSrc ?? '';
                    link.download = process.filename || 'captioned-video.mp4';
                    link.click();
                  }}
                  className="absolute bottom-3 left-3 z-30 inline-flex h-10 items-center sm:h-8 gap-1.5 rounded-full bg-black/60 px-3 text-xs font-medium text-white backdrop-blur-md transition-opacity duration-200 hover:bg-black/80 focus-visible:opacity-100 [@media(hover:hover)]:opacity-0 group-hover:opacity-100"
                  title="Download MP4"
                >
                  <Download className="h-3.5 w-3.5" />
                  <span>Download</span>
                </button>
              </>
            ) : failed ? (
              <div className="flex h-full w-full flex-col items-center justify-center gap-2 bg-destructive/15 p-4 text-center">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-destructive/20 text-destructive">
                  <AlertTriangle className="h-4 w-4" />
                </div>
                <div>
                  <p className="text-sm font-medium tracking-tight text-destructive">
                    Render Interrupted
                  </p>
                  <p className="mt-1 max-w-[240px] truncate text-xs text-muted-foreground">
                    {process.reason || 'Process failed during execution'}
                  </p>
                </div>
              </div>
            ) : (
              <div className="flex h-full w-full flex-col items-center justify-center gap-3 bg-[radial-gradient(60%_80%_at_50%_0%,rgb(124_108_255/0.28),transparent)] p-4 text-center">
                <div className="flex h-11 w-11 items-center justify-center rounded-full bg-white/10 text-white backdrop-blur">
                  <Loader2 className="h-5 w-5 animate-spin" />
                </div>
                <div>
                  <p className="text-sm font-medium tracking-tight text-white/90">
                    {process.stage === 'transcribing'
                      ? 'Transcribing audio…'
                      : 'Rendering captions…'}
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
          <div className="flex items-center justify-between p-4 pr-14">
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold tracking-tight text-foreground">
                {process.filename || process.id}
              </p>
              <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
                <span>
                  {formatDistanceToNow(new Date(process.createdAt), { addSuffix: true })}
                </span>
                <span>·</span>
                <span className="font-mono text-[10px] text-muted-foreground/80 ">
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
        className="absolute bottom-3.5 right-3.5 z-30 inline-flex h-10 w-10 items-center justify-center rounded-full border border-border bg-foreground/[0.05] text-muted-foreground sm:h-9 sm:w-9 transition-all duration-200 hover:border-destructive/40 hover:bg-destructive/15 hover:text-destructive focus-visible:opacity-100 [@media(hover:hover)]:opacity-0 group-hover:opacity-100"
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
        <p className="mt-3 text-sm text-muted-foreground">Loading videos…</p>
      </div>
    );
  }

  if (error && processes === null) {
    return (
      <Card className="flex flex-col items-center justify-center py-16 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-destructive/15 text-destructive">
          <AlertTriangle className="h-6 w-6" />
        </div>
        <h3 className="mt-4 text-base font-semibold tracking-tight">Could not load gallery</h3>
        <p className="mt-1 max-w-sm text-xs text-muted-foreground">{error}</p>
      </Card>
    );
  }

  if (processes !== null && processes.length === 0) {
    return (
      <Card className="flex flex-col items-center justify-center border-dashed px-6 py-20 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/15 text-primary">
          <Film className="h-7 w-7" />
        </div>
        <h3 className="mt-5 text-2xl font-semibold tracking-[-0.03em]">No captioned videos yet</h3>
        <p className="mt-2 max-w-sm text-sm leading-relaxed text-muted-foreground">
          Drop a video on the home page to reframe it and add subtitles.
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
      <div className="-mx-4 overflow-x-auto px-4 pb-1 sm:mx-0 sm:px-0">
        <div className="flex w-max items-center gap-2 sm:w-auto sm:flex-wrap">
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
                'inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium tracking-tight transition-all duration-200',
                filter === tab.id
                  ? 'border-transparent bg-foreground text-background'
                  : 'border-border bg-foreground/[0.03] text-muted-foreground hover:border-foreground/25 hover:text-foreground'
              )}
            >
              <span>{tab.label}</span>
              <span
                className={cn(
                  'rounded-full px-1.5 py-px font-mono text-[10px] tabular-nums',
                  filter === tab.id
                    ? 'bg-background/20 text-background'
                    : 'bg-foreground/10'
                )}
              >
                {tab.count}
              </span>
            </button>
          ))}
        </div>
      </div>

      {filteredProcesses.length === 0 ? (
        <div className="py-14 text-center text-xs text-muted-foreground">
          No videos match the selected &quot;{filter}&quot; filter.
        </div>
      ) : (
        <div className="grid grid-cols-[minmax(0,1fr)] gap-5 sm:grid-cols-2 lg:grid-cols-3 lg:gap-6">
          {filteredProcesses.map((process) => (
            <GalleryCard key={process.id} process={process} />
          ))}
        </div>
      )}
    </div>
  );
}
