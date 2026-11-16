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
  uploaded: 'bg-foreground/[0.06] text-muted-foreground',
  transcribing: 'bg-primary/15 text-primary',
  rendering: 'bg-[#22d3ee]/15 text-[#0891b2] dark:text-[#67e8f9]',
  done: 'bg-success/15 text-success',
  failed: 'bg-destructive/15 text-destructive',
};

export function StageBadge({ stage }: { stage: ProcessStage }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium tracking-tight ${STAGE_STYLE[stage]}`}
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
