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
  uploaded: 'border-border bg-card text-muted-foreground',
  transcribing: 'border-amber-400/40 bg-amber-400/10 text-amber-400',
  rendering: 'border-amber-400/40 bg-amber-400/10 text-amber-400',
  done: 'border-emerald-500/40 bg-emerald-500/10 text-emerald-400',
  failed: 'border-red-500/40 bg-red-500/10 text-red-400',
};

export function StageBadge({ stage }: { stage: ProcessStage }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium backdrop-blur ${STAGE_STYLE[stage]}`}
    >
      {stage === 'done' && <CheckCircle className="h-3.5 w-3.5" />}
      {stage === 'failed' && <XCircle className="h-3.5 w-3.5" />}
      {stage !== 'done' && stage !== 'failed' && (
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
      )}
      {STAGE_LABEL[stage]}
    </span>
  );
}
