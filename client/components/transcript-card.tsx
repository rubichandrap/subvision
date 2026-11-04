'use client';

import * as React from 'react';
import { Loader2, Plus, Trash2 } from 'lucide-react';

import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  fetchSegments,
  rerenderProcess,
  saveSegments,
  type ProcessStage,
  type Segment,
} from '@/lib/api';

// Transcript card on the Process detail page: every segment with its text
// and start/end time. Readable only after transcription — while
// transcribing it shows loading, never half-decoded text. Once rendering
// or done, text and per-segment timing are editable with client-side
// validation. It fetches on the stage the existing per-process poll
// reports, so no new polling mechanism.

export type DraftSegment = {
  id: string;
  sourceIndex: number | null;
  text: string;
  start: string;
  end: string;
};

export type SegmentError = {
  text?: string;
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

function formatInputSeconds(value: number): string {
  return String(Math.round(value * 1000) / 1000);
}

let draftRowSequence = 0;

function draftRowId(): string {
  draftRowSequence += 1;
  return `draft-row-${draftRowSequence}-${Date.now()}`;
}

function toDraft(segments: Segment[]): DraftSegment[] {
  return segments.map((s, index) => ({
    id: `source-${index}`,
    sourceIndex: index,
    text: s.text,
    start: formatInputSeconds(s.start),
    end: formatInputSeconds(s.end),
  }));
}

// Timestamp comparisons tolerate the floating-point noise produced by model
// inference and the millisecond rounding in these inputs. This mirrors the
// server's 2 ms tolerance so the editor does not reject a timeline the server
// accepts.
const timeEpsilon = 2e-3;

export function validateSegments(draft: DraftSegment[]): SegmentError[] {
  const parsed = draft.map((row) => ({
    start: Number(row.start),
    end: Number(row.end),
  }));
  return draft.map((row, i) => {
    const err: SegmentError = {};
    const { start, end } = parsed[i];
    if (row.text.trim() === '') {
      err.text = 'Enter the caption text.';
    }
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
        start < prev.end - timeEpsilon
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
  const [saving, setSaving] = React.useState(false);
  const [rerendering, setRerendering] = React.useState(false);
  const [savedNote, setSavedNote] = React.useState<string | null>(null);
  const [actionError, setActionError] = React.useState<string | null>(null);
  // Tracks unsaved edits so a poll-driven refetch cannot silently replace
  // what the user typed while the process was rendering.
  const dirtyRef = React.useRef(false);

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
        if (dirtyRef.current) return;
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

  const errors = validateSegments(draft);
  const hasErrors = errors.some((e) => e.text ?? e.start ?? e.end ?? e.order);

  const updateRow = (index: number, patch: Partial<DraftSegment>) => {
    dirtyRef.current = true;
    setDraft((prev) =>
      prev.map((row, i) => (i === index ? { ...row, ...patch } : row)),
    );
  };

  const addSegment = () => {
    dirtyRef.current = true;
    setActionError(null);
    setSavedNote(null);
    setDraft((prev) => {
      const parsedEnd = Number(prev[prev.length - 1]?.end ?? '0');
      const start = Number.isFinite(parsedEnd) ? parsedEnd : 0;
      return [
        ...prev,
        {
          id: draftRowId(),
          sourceIndex: null,
          // Empty text is deliberate: an unfilled manual cue must fail
          // validation rather than ship a placeholder into the Output.
          text: '',
          start: formatInputSeconds(start),
          end: formatInputSeconds(start + 2),
        },
      ];
    });
  };

  const removeSegment = (index: number) => {
    dirtyRef.current = true;
    setActionError(null);
    setSavedNote(null);
    setDraft((prev) => prev.filter((_, i) => i !== index));
  };

  // Edited rows merge back onto their source rows. Manual rows carry no
  // words; the server distributes timings across their token lengths.
  const toPayload = (): Segment[] =>
    draft.map((row) => {
      const source =
        row.sourceIndex === null || row.sourceIndex >= segments.length
          ? undefined
          : segments[row.sourceIndex];
      const text = row.text;
      const start =
        row.start.trim() !== ''
          ? Number(row.start)
          : (source?.start ?? 0);
      const end =
        row.end.trim() !== '' ? Number(row.end) : (source?.end ?? start);

      if (source) {
        return {
          ...source,
          text,
          start,
          end,
        };
      }

      return {
        text,
        start,
        end,
        words: [],
      };
    });

  const sourceIndices = (): number[] =>
    draft.map((row) =>
      row.sourceIndex === null || row.sourceIndex >= segments.length
        ? -1
        : row.sourceIndex,
    );

  const persistEdits = async (): Promise<Segment[]> => {
    const saved = await saveSegments(processId, toPayload(), sourceIndices());
    dirtyRef.current = false;
    setSegments(saved);
    setDraft(toDraft(saved));
    return saved;
  };

  const handleSave = async () => {
    setActionError(null);
    setSavedNote(null);
    setSaving(true);
    try {
      await persistEdits();
      setSavedNote('Edits saved. Reloading the page shows them.');
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const handleRerender = async () => {
    setActionError(null);
    setSavedNote(null);
    setRerendering(true);
    try {
      // Save first so the re-render carries exactly what is on screen;
      // the poll then moves the process through rendering to done.
      await persistEdits();
      await rerenderProcess(processId);
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : 'Re-render failed',
      );
    } finally {
      setRerendering(false);
    }
  };

  return (
    <Card className="border-2 p-5">
      <TranscriptHeader
        extra={
          <p className="text-xs text-muted-foreground">
            {draft.length} segment{draft.length === 1 ? '' : 's'} · times in
            seconds of the trimmed clip
          </p>
        }
      />
      <ol className="mt-4 space-y-4">
        {draft.map((row, i) => {
          const err = errors[i] ?? {};
          const invalid = Boolean(err.text ?? err.start ?? err.end ?? err.order);
          return (
            <li
              key={row.id}
              className={`border-2 p-3 ${invalid ? 'border-destructive/60' : 'border-border'}`}
            >
              <div className="flex items-center justify-between gap-2">
                <p className="font-mono text-xs font-bold uppercase tracking-wide text-muted-foreground">
                  Segment {i + 1}
                </p>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label={`Remove segment ${i + 1}`}
                  title="Remove segment"
                  disabled={saving || rerendering}
                  onClick={() => removeSegment(i)}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
              <Textarea
                aria-label={`Segment ${i + 1} text`}
                value={row.text}
                onChange={(e) => updateRow(i, { text: e.target.value })}
                className="mt-2 min-h-[60px]"
                aria-invalid={Boolean(err.text)}
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
              {(err.text ?? err.start ?? err.end ?? err.order) && (
                <div className="mt-2 space-y-0.5" role="alert">
                  {[err.text, err.start, err.end, err.order]
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
      <Button
        type="button"
        variant="outline"
        className="mt-4 w-full"
        disabled={saving || rerendering}
        onClick={addSegment}
      >
        <Plus className="mr-2 h-4 w-4" />
        Add segment
      </Button>
      {savedNote && (
        <p className="mt-4 border-2 border-border bg-[#7df29a] p-2 text-sm font-bold text-[#141414]">
          {savedNote}
        </p>
      )}
      {actionError && (
        <p
          className="mt-4 border-2 border-border bg-destructive p-2 text-sm font-bold text-destructive-foreground"
          role="alert"
        >
          {actionError}
        </p>
      )}
      <div className="mt-4 flex flex-wrap gap-2">
        <Button
          onClick={() => {
            void handleSave();
          }}
          disabled={saving || rerendering || hasErrors}
        >
          {saving && <Loader2 className="h-4 w-4 animate-spin" />}
          Save edits
        </Button>
        {stage === 'done' && (
          <Button
            variant="outline"
            onClick={() => {
              void handleRerender();
            }}
            disabled={saving || rerendering || hasErrors}
          >
            {rerendering && <Loader2 className="h-4 w-4 animate-spin" />}
            Save and re-render
          </Button>
        )}
      </div>
      {hasErrors && (
        <p className="mt-2 text-xs text-muted-foreground">
          Fix the timing errors above before saving.
        </p>
      )}
    </Card>
  );
}
