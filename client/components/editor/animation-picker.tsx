'use client';

import * as React from 'react';

import {
  ANIMATION_OPTIONS,
  MAX_WORDS_PER_PAGE,
  MIN_WORDS_PER_PAGE,
  type AnimationChoice,
} from '@/lib/edit-spec';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';

// The Animation panel: one chip per subtitle animation plus the dice. The
// hint under the chips describes the highlighted choice; "random" resolves
// to a concrete animation when the video is generated. The words-per-page
// slider shows only when it matters (pop, karaoke, random) — fade and slide
// render full segments, so the knob would have no effect there.

export interface AnimationPickerProps {
  value: AnimationChoice;
  wordsPerPage: number;
  onChange: (value: AnimationChoice) => void;
  onWordsPerPageChange: (wordsPerPage: number) => void;
  disabled?: boolean;
}

export function AnimationPicker({
  value,
  wordsPerPage,
  onChange,
  onWordsPerPageChange,
  disabled,
}: AnimationPickerProps) {
  const hint = ANIMATION_OPTIONS.find((option) => option.value === value)?.hint;
  const pagingMatters = value === 'pop' || value === 'karaoke' || value === 'random';

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-2">
        {ANIMATION_OPTIONS.map((option) => {
          const active = value === option.value;
          return (
            <button
              key={option.value}
              type="button"
              disabled={disabled}
              onClick={() => onChange(option.value)}
              className={`border-2 border-border px-3 py-2.5 text-left text-sm font-bold shadow-brutal-sm transition-all duration-100 disabled:opacity-50 disabled:shadow-none ${
                active
                  ? 'bg-primary text-primary-foreground -translate-x-0.5 -translate-y-0.5 shadow-brutal'
                  : 'bg-card text-muted-foreground hover:-translate-x-0.5 hover:-translate-y-0.5 hover:bg-accent hover:text-accent-foreground hover:shadow-brutal'
              }`}
            >
              {option.value === 'random' ? 'Random' : option.label}
            </button>
          );
        })}
      </div>
      {hint && <p className="text-xs leading-relaxed text-muted-foreground">{hint}</p>}
      {pagingMatters && (
        <div>
          <div className="flex items-baseline justify-between">
            <Label className="text-sm">Words per page</Label>
            <span className="text-xs tabular-nums text-muted-foreground">{wordsPerPage}</span>
          </div>
          <div className="mt-2">
            <Slider
              value={[wordsPerPage]}
              min={MIN_WORDS_PER_PAGE}
              max={MAX_WORDS_PER_PAGE}
              step={1}
              disabled={disabled}
              aria-label="Words per caption page"
              onValueChange={([next]) => next !== undefined && onWordsPerPageChange(next)}
            />
          </div>
        </div>
      )}
    </div>
  );
}
