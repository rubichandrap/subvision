'use client';

import * as React from 'react';
import Link from 'next/link';
import { formatDistanceToNow } from 'date-fns';
import { FileVideo2, Loader2, Trash2 } from 'lucide-react';

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
import { downloadUrl, type Process } from '@/lib/api';

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

  return (
    <div className="group relative">
      <Link href={`/processes/${process.id}`} className="block">
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
          <div className="p-3 pr-10">
            <p className="truncate text-sm font-medium">{process.filename || process.id}</p>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {formatDistanceToNow(new Date(process.createdAt), { addSuffix: true })}
            </p>
          </div>
        </Card>
      </Link>
      <DeleteProcessButton
        process={process}
        className="absolute bottom-3 right-3 inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground/70 opacity-0 transition-opacity hover:bg-red-500/10 hover:text-red-400 focus-visible:opacity-100 group-hover:opacity-100"
        label={`Delete ${process.filename || process.id}`}
      />
    </div>
  );
}

export function ProcessGallery() {
  const { processes, error } = useProcessList();

  if (processes === null && !error) {
    return (
      <div className="flex items-center justify-center py-24">
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
        <FileVideo2 className="mb-4 h-10 w-10 text-muted-foreground/60" />
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
