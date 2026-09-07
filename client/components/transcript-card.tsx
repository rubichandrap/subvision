'use client';

import * as React from 'react';
import { Loader2 } from 'lucide-react';

import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { fetchSegments, type ProcessStage, type Segment } from '@/lib/api';

// Transcript card on the Process detail page: every segment with its text
// and start/end time. Readable only after transcription — while
// transcribing it shows loading, never half-decoded text. Once rendering
// or done, text and per-segment timing are editable with client-side
// validation. It fetches on the stage the existing per-process poll
// reports, so no new polling mechanism.

export type DraftSegment = {
  text: string;
  start: string;
  end: string;
};

export type SegmentError = {
  start?: string;
  end?: string;
  order?: string;
};

function TranscriptHeader({ extra }: { extra?: React.ReactNode }) {
  if (extra) {
    return (
      <div className="flex items-baseline justify-between gap-3">
        <p className="font-mono text-sm font-bold uppercase tracking-wide">
          Transcript
        </p>
        {extra}
      </div>
    );
  }
  return (
    <p className="font-mono text-sm font-bold uppercase tracking-wide">
      Transcript
    </p>
  );
}

function toDraft(segments: Segment[]): DraftSegment[] {
  return segments.map((s) => ({
    text: s.text,
    start: String(s.start),
    end: String(s.end),
  }));
}

export function validateSegments(draft: DraftSegment[]): SegmentError[] {
  const parsed = draft.map((row) => ({
    start: Number(row.start),
    end: Number(row.end),
  }));
  return draft.map((row, i) => {
    const err: SegmentError = {};
    const { start, end } = parsed[i];
    if (row.start.trim() === '' || !Number.isFinite(start)) {
      err.start = 'Enter a start time in seconds.';
    } else if (start < 0) {
      err.start = 'Start time cannot be negative.';
    }
    if (row.end.trim() === '' || !Number.isFinite(end)) {
      err.end = 'Enter an end time in seconds.';
    } else if (
      Number.isFinite(start) &&
      row.start.trim() !== '' &&
      end <= start
    ) {
      err.end = 'End time must be after start time.';
    }
    if (i > 0) {
      const prev = parsed[i - 1];
      if (
        Number.isFinite(start) &&
        Number.isFinite(prev.end) &&
        row.start.trim() !== '' &&
        draft[i - 1].end.trim() !== '' &&
        start < prev.end
      ) {
        err.order = 'Segments must stay in order, without overlap.';
      }
    }
    return err;
  });
}

export function TranscriptCard({
  processId,
  stage,
}: {
  processId: string;
  stage: ProcessStage;
}) {
  const [segments, setSegments] = React.useState<Segment[] | null>(null);
  const [fetchError, setFetchError] = React.useState<string | null>(null);
  const [draft, setDraft] = React.useState<DraftSegment[]>([]);

  // Refetch only when the polled stage moves (or the process changes) —
  // the existing per-process poll drives updates, nothing new polls here.
  React.useEffect(() => {
    if (stage !== 'rendering' && stage !== 'done') return;
    let cancelled = false;
    setSegments(null);
    setFetchError(null);
    fetchSegments(processId).then(
      (segs) => {
        if (cancelled) return;
        setSegments(segs);
        setDraft(toDraft(segs));
      },
      (err: unknown) => {
        if (cancelled) return;
        setFetchError(
          err instanceof Error ? err.message : 'Failed to load transcript',
        );
      },
    );
    return () => {
      cancelled = true;
    };
  }, [processId, stage]);

  if (stage === 'transcribing') {
    return (
      <Card className="border-2 p-5">
        <TranscriptHeader />
        <div className="flex items-center gap-2 py-6 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          Transcribing… the transcript appears once transcription finishes.
        </div>
      </Card>
    );
  }

  if (fetchError) {
    return (
      <Card className="border-2 p-5">
        <TranscriptHeader />
        <p className="py-4 text-sm text-muted-foreground">{fetchError}</p>
      </Card>
    );
  }

  if (segments === null) {
    return (
      <Card className="border-2 p-5">
        <TranscriptHeader />
        <div className="flex items-center gap-2 py-6 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading transcript…
        </div>
      </Card>
    );
  }

  if (segments.length === 0) {
    return (
      <Card className="border-2 p-5">
        <TranscriptHeader />
        <p className="py-4 text-sm text-muted-foreground">
          No transcript segments for this process yet.
        </p>
      </Card>
    );
  }

  const errors = validateSegments(draft);

  const updateRow = (index: number, patch: Partial<DraftSegment>) => {
    setDraft((prev) =>
      prev.map((row, i) => (i === index ? { ...row, ...patch } : row)),
    );
  };

  return (
    <Card className="border-2 p-5">
      <TranscriptHeader
        extra={
          <p className="text-xs text-muted-foreground">
            {draft.length} segment{draft.length === 1 ? '' : 's'} · times in
            seconds
          </p>
        }
      />
      <ol className="mt-4 space-y-4">
        {draft.map((row, i) => {
          const err = errors[i] ?? {};
          const invalid = Boolean(err.start ?? err.end ?? err.order);
          return (
            <li
              key={i}
              className={`border-2 p-3 ${invalid ? 'border-destructive/60' : 'border-border'}`}
            >
              <p className="font-mono text-xs font-bold uppercase tracking-wide text-muted-foreground">
                Segment {i + 1}
              </p>
              <Textarea
                aria-label={`Segment ${i + 1} text`}
                value={row.text}
                onChange={(e) => updateRow(i, { text: e.target.value })}
                className="mt-2 min-h-[60px]"
              />
              <div className="mt-2 grid grid-cols-2 gap-2">
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-muted-foreground">
                    Start (s)
                  </span>
                  <Input
                    aria-label={`Segment ${i + 1} start time in seconds`}
                    inputMode="decimal"
                    value={row.start}
                    onChange={(e) => updateRow(i, { start: e.target.value })}
                    aria-invalid={Boolean(err.start ?? err.order)}
                  />
                </label>
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-muted-foreground">
                    End (s)
                  </span>
                  <Input
                    aria-label={`Segment ${i + 1} end time in seconds`}
                    inputMode="decimal"
                    value={row.end}
                    onChange={(e) => updateRow(i, { end: e.target.value })}
                    aria-invalid={Boolean(err.end ?? err.order)}
                  />
                </label>
              </div>
              {(err.start ?? err.end ?? err.order) && (
                <div className="mt-2 space-y-0.5" role="alert">
                  {[err.start, err.end, err.order]
                    .filter(Boolean)
                    .map((message) => (
                      <p key={message} className="text-xs font-medium text-destructive">
                        {message}
                      </p>
                    ))}
                </div>
              )}
            </li>
          );
        })}
      </ol>
    </Card>
  );
}
