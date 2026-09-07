'use client';

import * as React from 'react';
import { RotateCcw } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Slider } from '@/components/ui/slider';
import { FRAME_PRESETS, type FrameState } from '@/lib/edit-spec';

// The Frame panel: platform presets, the free ratio (dragged on the preview
// itself), and the cover zoom. Panning happens by dragging the video.

export interface FramePickerProps {
  frame: FrameState;
  onChange: (frame: FrameState) => void;
  disabled?: boolean;
}

export function FramePicker({ frame, onChange, disabled }: FramePickerProps) {
  const ratioLabel = frame.preset === 'free' ? `${frame.ratio.toFixed(2)} : 1` : frame.preset;

  return (
    <div className="space-y-5">
      <div>
        <p className="text-sm font-medium">Frame</p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          The canvas the video is cropped to fill.
        </p>
        <div className="mt-3 grid grid-cols-5 gap-2">
          {FRAME_PRESETS.map((preset) => {
            const active = frame.preset === preset.name;
            return (
              <button
                key={preset.name}
                type="button"
                disabled={disabled}
                title={preset.label}
                onClick={() =>
                  onChange({ ...frame, preset: preset.name, ratio: preset.ratio })
                }
                className={`flex flex-col items-center gap-1.5 border-2 border-border px-1 py-2.5 text-xs font-bold shadow-brutal-sm transition-all duration-100 disabled:opacity-50 disabled:shadow-none ${
                  active
                    ? 'bg-primary text-primary-foreground -translate-x-0.5 -translate-y-0.5 shadow-brutal'
                    : 'bg-card text-muted-foreground hover:-translate-x-0.5 hover:-translate-y-0.5 hover:bg-accent hover:text-accent-foreground hover:shadow-brutal'
                }`}
              >
                <span
                  className={`block rounded-sm border-current ${
                    active ? 'border-2' : 'border'
                  }`}
                  style={{
                    width: preset.ratio >= 1 ? 22 : 22 * preset.ratio,
                    height: preset.ratio >= 1 ? 22 / preset.ratio : 22,
                  }}
                />
                {preset.name}
              </button>
            );
          })}
          <button
            type="button"
            disabled={disabled}
            onClick={() => onChange({ ...frame, preset: 'free' })}
            className={`flex flex-col items-center gap-1.5 border-2 border-border px-1 py-2.5 text-xs font-bold shadow-brutal-sm transition-all duration-100 disabled:opacity-50 disabled:shadow-none ${
              frame.preset === 'free'
                ? 'bg-primary text-primary-foreground -translate-x-0.5 -translate-y-0.5 shadow-brutal'
                : 'bg-card text-muted-foreground hover:-translate-x-0.5 hover:-translate-y-0.5 hover:bg-accent hover:text-accent-foreground hover:shadow-brutal'
            }`}
          >
            <svg viewBox="0 0 22 22" className="h-[22px] w-[22px]" fill="none" stroke="currentColor" strokeWidth="1.6">
              <path d="M4 8V4h4M18 8V4h-4M4 14v4h4M18 14v4h-4" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
            Free
          </button>
        </div>
        {frame.preset === 'free' && (
          <p className="mt-2 text-xs text-muted-foreground">
            Drag the handle in the preview’s corner to set any ratio. Currently{' '}
            <span className="font-medium text-foreground">{ratioLabel}</span>.
          </p>
        )}
      </div>

      <div>
        <div className="flex items-baseline justify-between">
          <p className="text-sm font-medium">Zoom</p>
          <span className="text-xs tabular-nums text-muted-foreground">
            {frame.zoom.toFixed(2)}×
          </span>
        </div>
        <Slider
          className="mt-2.5"
          value={[frame.zoom]}
          min={1}
          max={5}
          step={0.05}
          disabled={disabled}
          aria-label="Zoom"
          onValueChange={([zoom]) => onChange({ ...frame, zoom })}
        />
        <p className="mt-1.5 text-xs text-muted-foreground">
          Zoom past the frame, then drag the video to choose what stays visible.
        </p>
      </div>

      <Button
        variant="ghost"
        size="sm"
        disabled={disabled}
        className="text-muted-foreground"
        onClick={() =>
          onChange({ preset: '9:16', ratio: 9 / 16, zoom: 1, panX: 0, panY: 0 })
        }
      >
        <RotateCcw className="h-3.5 w-3.5" />
        Reset frame
      </Button>
    </div>
  );
}
