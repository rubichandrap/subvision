'use client';

import * as React from 'react';

import { ANIMATION_OPTIONS, type AnimationChoice } from '@/lib/edit-spec';

// The Animation panel: one chip per subtitle animation plus the dice. The
// hint under the chips describes the highlighted choice; "random" resolves
// to a concrete animation when the video is generated.

export interface AnimationPickerProps {
  value: AnimationChoice;
  onChange: (value: AnimationChoice) => void;
  disabled?: boolean;
}

export function AnimationPicker({ value, onChange, disabled }: AnimationPickerProps) {
  const hint = ANIMATION_OPTIONS.find((option) => option.value === value)?.hint;

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-1.5">
        {ANIMATION_OPTIONS.map((option) => {
          const active = value === option.value;
          return (
            <button
              key={option.value}
              type="button"
              disabled={disabled}
              onClick={() => onChange(option.value)}
              className={`rounded-lg border px-3 py-2.5 text-left text-sm font-medium transition-colors disabled:opacity-50 ${
                active
                  ? 'border-primary bg-primary/15 text-primary'
                  : 'border-border bg-card text-muted-foreground hover:border-primary/40 hover:text-foreground'
              }`}
            >
              {option.value === 'random' ? 'Random 🎲' : option.label}
            </button>
          );
        })}
      </div>
      {hint && <p className="text-xs leading-relaxed text-muted-foreground">{hint}</p>}
    </div>
  );
}
