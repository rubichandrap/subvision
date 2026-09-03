'use client';

import * as React from 'react';
import Link from 'next/link';
import { formatDistanceToNow } from 'date-fns';
import {
  ArrowLeft,
  Check,
  CheckCircle,
  Copy,
  Download,
  FileVideo2,
  Loader2,
  Trash2,
  XCircle,
} from 'lucide-react';

import { STAGE_LABEL } from '@/components/process-stage';
import { DeleteProcessDialog } from '@/components/process-gallery';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { useRouter } from 'next/navigation';
import { useProcess } from '@/hooks/use-process-polling';
import { IN_FLIGHT_STAGES, downloadUrl, type ProcessStage } from '@/lib/api';

// One Process, live: a horizontal stepper over the real server-side stages,
// the rendered Output once it exists, and the download behind it.

const PIPELINE_STAGES: ProcessStage[] = [
  'uploaded',
  'transcribing',
  'rendering',
  'done',
];

export function ProcessDetails({ processId }: { processId: string }) {
  const { process, notFound, error } = useProcess(processId);
  const router = useRouter();
  const [deleteOpen, setDeleteOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);

  const handleCopyId = async () => {
    if (!process) return;
    try {
      await navigator.clipboard.writeText(process.id);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard unavailable (insecure origin) — nothing sensible to do
    }
  };

  const handleDownload = () => {
    if (!process?.downloadUrl) return;
    window.location.href = downloadUrl(process);
  };

  if (notFound || (error && !process)) {
    return (
      <Card className="flex flex-col items-center justify-center border-border/70 bg-card/50 py-16">
        <FileVideo2 className="mb-4 h-10 w-10 text-muted-foreground/60" />
        <h3 className="font-display text-lg font-semibold">
          {notFound ? 'Process not found' : 'Could not load this process'}
        </h3>
        <p className="mt-2 text-center text-sm text-muted-foreground">
          {notFound
            ? 'The process you are looking for does not exist.'
            : error}
        </p>
        <Button asChild className="mt-6">
          <Link href="/processes">Back to the gallery</Link>
        </Button>
      </Card>
    );
  }

  if (!process) {
    return (
      <div className="flex justify-center items-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const inFlight = IN_FLIGHT_STAGES.has(process.stage);
  const currentStageIndex = PIPELINE_STAGES.indexOf(process.stage);
  const done = process.stage === 'done' && Boolean(process.downloadUrl);

  return (
    <div className="space-y-5">
      <Link
        href="/processes"
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to the gallery
      </Link>

      <Card className="border-border/70 bg-card/50 p-5">
        <div className="flex flex-wrap items-center gap-3">
          <div className="min-w-0 flex-1">
            <h2 className="truncate font-display text-xl font-semibold">
              {process.filename || process.id}
            </h2>
            <div className="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
              <span>{formatDistanceToNow(new Date(process.createdAt), { addSuffix: true })}</span>
              <span className="h-1 w-1 rounded-full bg-border" aria-hidden="true" />
              <span className="inline-flex min-w-0 items-center gap-1.5">
                <code
                  title={process.id}
                  className="max-w-[180px] truncate rounded bg-muted px-1.5 py-0.5 font-mono text-[11px] leading-none text-muted-foreground sm:max-w-[320px] md:max-w-[420px]"
                >
                  {process.id}
                </code>
                <button
                  type="button"
                  aria-label="Copy process id"
                  className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded border border-transparent text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                  onClick={() => {
                    void handleCopyId();
                  }}
                >
                  {copied ? (
                    <Check className="h-3.5 w-3.5 text-emerald-500" />
                  ) : (
                    <Copy className="h-3.5 w-3.5" />
                  )}
                </button>
              </span>
            </div>
          </div>
          <div className="ml-auto flex items-center gap-2">
            {done && (
              <Button onClick={handleDownload}>
                <Download className="h-4 w-4" />
                Download
              </Button>
            )}
            <Button
              variant="outline"
              className="border-red-500/30 text-red-400 hover:bg-red-500/10 hover:text-red-400"
              onClick={() => setDeleteOpen(true)}
            >
              <Trash2 className="h-4 w-4" />
              Delete
            </Button>
          </div>
        </div>

        {/* Horizontal stepper over the real stages */}
        <div className="mt-6 flex items-center">
          {PIPELINE_STAGES.map((stage, index) => {
            const isCurrent = stage === process.stage;
            const reached = currentStageIndex >= index;
            const isLast = index === PIPELINE_STAGES.length - 1;
            return (
              <React.Fragment key={stage}>
                <div className="flex flex-col items-center gap-1.5">
                  <div
                    className={`flex h-8 w-8 items-center justify-center rounded-full border ${
                      isCurrent && inFlight
                        ? 'border-amber-400/60 bg-amber-400/10 text-amber-400'
                        : isCurrent && process.stage === 'failed'
                          ? 'border-red-500/60 bg-red-500/10 text-red-500'
                          : reached
                            ? 'border-primary/60 bg-primary/15 text-primary'
                            : 'border-border bg-card text-muted-foreground/50'
                    }`}
                  >
                    {isCurrent && inFlight ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : isCurrent && process.stage === 'failed' ? (
                      <XCircle className="h-4 w-4" />
                    ) : reached ? (
                      <CheckCircle className="h-4 w-4" />
                    ) : (
                      <span className="text-xs">{index + 1}</span>
                    )}
                  </div>
                  <span
                    className={`text-[11px] font-medium ${
                      reached ? 'text-foreground' : 'text-muted-foreground/60'
                    }`}
                  >
                    {STAGE_LABEL[stage]}
                  </span>
                </div>
                {!isLast && (
                  <div
                    className={`mx-2 mb-5 h-px flex-1 ${
                      currentStageIndex > index ? 'bg-primary/60' : 'bg-border'
                    }`}
                  />
                )}
              </React.Fragment>
            );
          })}
        </div>

        {process.stage === 'failed' && process.reason && (
          <div className="mt-5 rounded-lg border border-red-500/30 bg-red-500/10 p-3.5 text-sm text-red-400">
            This process failed: {process.reason}
          </div>
        )}
      </Card>

      {done ? (
        <Card className="overflow-hidden border-border/70 bg-card/50 p-0">
          <div className="flex items-center justify-between border-b border-border/60 px-5 py-3.5">
            <p className="font-display text-sm font-semibold">Your captioned video</p>
            <Badge variant="secondary" className="font-normal">
              ready to post
            </Badge>
          </div>
          <video
            src={downloadUrl(process)}
            controls
            playsInline
            className="max-h-[65vh] w-full bg-black object-contain"
          />
        </Card>
      ) : inFlight ? (
        <Card className="flex flex-col items-center justify-center border-border/70 bg-card/50 py-14">
          <Loader2 className="h-8 w-8 animate-spin text-amber-400" />
          <p className="mt-4 font-display font-semibold">
            {STAGE_LABEL[process.stage]}…
          </p>
          <p className="mt-1.5 text-sm text-muted-foreground">
            This page updates itself. No need to refresh.
          </p>
        </Card>
      ) : null}

      {process && (
        <DeleteProcessDialog
          process={process}
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
          onDeleted={() => router.push('/processes')}
        />
      )}
    </div>
  );
}
