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
    'bg-muted text-muted-foreground',
  transcribing:
    'bg-primary text-primary-foreground',
  rendering:
    'bg-accent text-accent-foreground',
  done:
    'bg-[#7df29a] text-[#141414]',
  failed:
    'bg-destructive text-destructive-foreground',
};

export function StageBadge({ stage }: { stage: ProcessStage }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 border-2 border-border px-2.5 py-0.5 font-mono text-[11px] font-bold uppercase tracking-wide shadow-brutal-sm ${STAGE_STYLE[stage]}`}
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
