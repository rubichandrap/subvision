'use client';

import * as React from 'react';
import { Pause, Play } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Slider } from '@/components/ui/slider';
import { formatTime, type TrimState } from '@/lib/edit-spec';

// The duration controls: a dual-handle window over the video's timeline with
// a looped preview inside the window. Trimming happens server-side; this is
// the honest preview of it.

export interface TrimTimelineProps {
  duration: number;
  trim: TrimState;
  playhead: number;
  videoRef: React.RefObject<HTMLVideoElement | null>;
  onTrimChange: (trim: TrimState) => void;
  disabled?: boolean;
}

export function TrimTimeline({
  duration,
  trim,
  playhead,
  videoRef,
  onTrimChange,
  disabled,
}: TrimTimelineProps) {
  const [playing, setPlaying] = React.useState(false);

  const seek = (time: number) => {
    const video = videoRef.current;
    if (video) video.currentTime = time;
  };

  const togglePlay = () => {
    const video = videoRef.current;
    if (!video) return;
    if (video.paused) {
      if (video.currentTime < trim.start || video.currentTime >= trim.end) {
        video.currentTime = trim.start;
      }
      void video.play();
    } else {
      video.pause();
    }
  };

  React.useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    const onPlay = () => setPlaying(true);
    const onPause = () => setPlaying(false);
    video.addEventListener('play', onPlay);
    video.addEventListener('pause', onPause);
    return () => {
      video.removeEventListener('play', onPlay);
      video.removeEventListener('pause', onPause);
    };
  }, [videoRef]);

  const handles: [number, number] = [trim.start, trim.end];
  const selected = trim.end > trim.start;

  return (
    <div className="rounded-xl border border-border/70 bg-card/50 p-4">
      <div className="flex items-center gap-3">
        <Button
          size="icon"
          variant="secondary"
          aria-label={playing ? 'Pause preview' : 'Play preview inside the trim'}
          onClick={togglePlay}
          disabled={disabled || !selected}
        >
          {playing ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
        </Button>
        <div className="min-w-0 flex-1">
          <Slider
            value={handles}
            min={0}
            max={Math.max(duration, 0.1)}
            step={0.1}
            disabled={disabled || duration <= 0}
            aria-label="Trim window"
            onValueChange={([start, end]) => {
              if (end - start < 0.2) {
                // Keep a minimal, still-meaningful window.
                if (start !== trim.start) {
                  onTrimChange({ start, end: Math.min(duration, start + 0.2) });
                } else {
                  onTrimChange({ start: Math.max(0, end - 0.2), end });
                }
                return;
              }
              onTrimChange({ start, end });
              if (playhead < start || playhead > end) seek(start);
            }}
          />
          <div className="mt-2 flex justify-between text-xs tabular-nums text-muted-foreground">
            <span>{formatTime(trim.start)}</span>
            <span className="font-medium text-foreground">{formatTime(playhead)}</span>
            <span>{selected ? formatTime(trim.end) : formatTime(duration)}</span>
          </div>
        </div>
        <div className="hidden text-right text-xs text-muted-foreground sm:block">
          <p className="font-medium text-foreground">
            {selected ? formatTime(trim.end - trim.start) : formatTime(duration)}
          </p>
          <p>{selected ? 'kept' : 'full length'}</p>
        </div>
      </div>
    </div>
  );
}
