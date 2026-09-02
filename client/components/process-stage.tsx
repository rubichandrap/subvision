import { CheckCircle, Loader2, XCircle } from 'lucide-react';

import type { ProcessStage } from '@/lib/api';

export const STAGE_LABEL: Record<ProcessStage, string> = {
  uploaded: 'Uploaded',
  transcribing: 'Transcribing',
  rendering: 'Rendering',
  done: 'Completed',
  failed: 'Failed',
};

export function StageBadge({ stage }: { stage: ProcessStage }) {
  if (stage === 'done') {
    return (
      <div className="flex items-center text-green-500 text-sm font-medium">
        <CheckCircle className="w-4 h-4 mr-1" />
        {STAGE_LABEL.done}
      </div>
    );
  }
  if (stage === 'failed') {
    return (
      <div className="flex items-center text-red-500 text-sm font-medium">
        <XCircle className="w-4 h-4 mr-1" />
        {STAGE_LABEL.failed}
      </div>
    );
  }
  return (
    <div className="flex items-center text-amber-500 text-sm font-medium">
      <Loader2 className="w-4 h-4 mr-1 animate-spin" />
      {STAGE_LABEL[stage]}
    </div>
  );
}
