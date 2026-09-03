import { CheckCircle, Loader2, XCircle } from 'lucide-react';

import type { ProcessStage } from '@/lib/api';

export const STAGE_LABEL: Record<ProcessStage, string> = {
  uploaded: 'Uploaded',
  transcribing: 'Transcribing',
  rendering: 'Rendering',
  done: 'Completed',
  failed: 'Failed',
};

const STAGE_STYLE: Record<ProcessStage, string> = {
  uploaded:
    'border-border/80 bg-muted/80 text-muted-foreground shadow-2xs',
  transcribing:
    'border-amber-500/40 bg-amber-50 text-amber-800 dark:bg-amber-500/15 dark:border-amber-400/40 dark:text-amber-300 shadow-2xs',
  rendering:
    'border-amber-500/40 bg-amber-50 text-amber-800 dark:bg-amber-500/15 dark:border-amber-400/40 dark:text-amber-300 shadow-2xs',
  done:
    'border-emerald-500/40 bg-emerald-50 text-emerald-800 dark:bg-emerald-500/15 dark:border-emerald-500/40 dark:text-emerald-300 shadow-2xs',
  failed:
    'border-rose-500/40 bg-rose-50 text-rose-800 dark:bg-rose-500/15 dark:border-rose-500/40 dark:text-rose-300 shadow-2xs',
};

export function StageBadge({ stage }: { stage: ProcessStage }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-semibold tracking-wide backdrop-blur-md transition-colors ${STAGE_STYLE[stage]}`}
    >
      {stage === 'done' && <CheckCircle className="h-3 w-3" />}
      {stage === 'failed' && <XCircle className="h-3 w-3" />}
      {stage !== 'done' && stage !== 'failed' && (
        <Loader2 className="h-3 w-3 animate-spin" />
      )}
      {STAGE_LABEL[stage]}
    </span>
  );
}
