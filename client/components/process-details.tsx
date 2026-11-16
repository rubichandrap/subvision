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
import { TranscriptCard } from '@/components/transcript-card';
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
    window.location.href = downloadUrl(process, { download: true });
  };

  if (notFound || (error && !process)) {
    return (
      <Card className="flex flex-col items-center justify-center px-6 py-16">
        <FileVideo2 className="mb-4 h-10 w-10 text-muted-foreground/60" />
        <h3 className="text-2xl font-semibold tracking-[-0.03em]">
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
    <div className="space-y-5 rise">
      <Link
        href="/processes"
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to the gallery
      </Link>

      <Card className="p-5 sm:p-7">
        <div className="flex flex-wrap items-center gap-x-5 gap-y-3">
          <div className="min-w-0 max-w-xl flex-1">
            <h2 className="truncate text-2xl font-semibold tracking-[-0.03em] sm:text-3xl">
              {process.filename || process.id}
            </h2>
            <div className="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
              <span>{formatDistanceToNow(new Date(process.createdAt), { addSuffix: true })}</span>
              <span className="h-1 w-1 rounded-full bg-border" aria-hidden="true" />
              <span className="inline-flex min-w-0 items-center gap-1.5">
                <code
                  title={process.id}
                  className="max-w-[180px] truncate rounded-md bg-foreground/[0.06] px-1.5 py-0.5 font-mono text-[11px] leading-none text-muted-foreground sm:max-w-[320px] md:max-w-[420px]"
                >
                  {process.id}
                </code>
                <button
                  type="button"
                  aria-label="Copy process id"
                  className="inline-flex h-9 w-9 shrink-0 items-center sm:h-7 sm:w-7 justify-center rounded-md text-muted-foreground transition-colors hover:bg-foreground/10 hover:text-foreground"
                  onClick={() => {
                    void handleCopyId();
                  }}
                >
                  {copied ? (
                    <Check className="h-3.5 w-3.5 text-success" />
                  ) : (
                    <Copy className="h-3.5 w-3.5" />
                  )}
                </button>
              </span>
            </div>
          </div>
          <div className="flex w-full items-center gap-2 sm:ml-auto sm:w-auto">
            {done && (
              <Button onClick={handleDownload}>
                <Download className="h-4 w-4" />
                Download
              </Button>
            )}
            <Button
              variant="outline"
              className="border-destructive/30 text-destructive hover:border-destructive/50 hover:bg-destructive/10 hover:text-destructive"
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
                    className={`flex h-9 w-9 items-center justify-center rounded-full border border-transparent transition-colors duration-300 ${
                      isCurrent && inFlight
                        ? 'bg-primary text-primary-foreground shadow-[0_0_0_6px_rgb(124_108_255/0.18)]'
                        : isCurrent && process.stage === 'failed'
                          ? 'bg-destructive text-destructive-foreground'
                          : reached
                            ? 'bg-success/20 text-success'
                            : 'border-border bg-foreground/[0.04] text-muted-foreground'
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
                    className={`text-[11px] font-medium sm:text-xs ${
                      reached ? 'text-foreground' : 'text-muted-foreground/60'
                    }`}
                  >
                    {STAGE_LABEL[stage]}
                  </span>
                </div>
                {!isLast && (
                  <div
                    className={`mx-1 mb-6 h-0.5 min-w-2 flex-1 sm:mx-2 rounded-full transition-colors duration-500 ${
                      currentStageIndex > index ? 'bg-brand' : 'bg-foreground/10'
                    }`}
                  />
                )}
              </React.Fragment>
            );
          })}
        </div>

        {process.stage === 'failed' && process.reason && (
          <div className="mt-5 rounded-xl bg-destructive/15 p-4 text-sm font-medium text-destructive">
            This process failed: {process.reason}
          </div>
        )}
      </Card>

      {(process.stage === 'transcribing' ||
        process.stage === 'rendering' ||
        process.stage === 'done') && (
        <TranscriptCard processId={process.id} stage={process.stage} />
      )}

      {done ? (
        <Card className="overflow-hidden p-0">
          <div className="flex items-center justify-between border-b border-border px-5 py-4">
            <p className="text-sm font-medium tracking-tight">Your captioned video</p>
          </div>
          <video
            src={downloadUrl(process)}
            controls
            playsInline
            className="max-h-[65vh] w-full bg-black object-contain"
          />
        </Card>
      ) : inFlight ? (
        <Card className="flex flex-col items-center justify-center px-6 py-14">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
          <p className="mt-4 text-lg font-semibold tracking-tight">
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
