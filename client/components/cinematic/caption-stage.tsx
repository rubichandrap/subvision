'use client';

import * as React from 'react';

import { cn } from '@/lib/utils';

// A live miniature of what Subvision makes: a letterboxed frame of moving
// colour with karaoke captions landing word by word over a ticking timecode.
// It is a demonstration, not footage; the real preview lives in the editor.

const PHRASES = [
  ['Captions', 'that', 'look', 'directed'],
  ['Every', 'word', 'lands', 'on', 'time'],
  ['Trim.', 'Reframe.', 'Export.'],
];

const WORD_MS = 430;
const HOLD_MS = 1100;

function timecode(ms: number) {
  const total = Math.floor(ms / 1000);
  const frames = Math.floor(((ms % 1000) / 1000) * 24);
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${pad(Math.floor(total / 3600))}:${pad(Math.floor(total / 60) % 60)}:${pad(total % 60)}:${pad(frames)}`;
}

export function CaptionStage({ className }: { className?: string }) {
  const [phrase, setPhrase] = React.useState(0);
  const [word, setWord] = React.useState(-1);
  const [clock, setClock] = React.useState(0);

  React.useEffect(() => {
    const start = performance.now();
    const tick = window.setInterval(() => setClock(performance.now() - start), 83);
    return () => window.clearInterval(tick);
  }, []);

  React.useEffect(() => {
    const words = PHRASES[phrase];
    const timer = window.setTimeout(
      () => {
        if (word < words.length - 1) {
          setWord(word + 1);
        } else {
          setPhrase((phrase + 1) % PHRASES.length);
          setWord(-1);
        }
      },
      word === words.length - 1 ? HOLD_MS : word === -1 ? 380 : WORD_MS
    );
    return () => window.clearTimeout(timer);
  }, [phrase, word]);

  const words = PHRASES[phrase];
  const progress = Math.min(1, (word + 1) / words.length);

  return (
    <div
      className={cn(
        'relative isolate aspect-[4/5] w-full overflow-hidden rounded-[1.75rem] border border-white/10 bg-[#07060d] shadow-soft-lg sm:aspect-[16/9] lg:aspect-[2.1/1]',
        className
      )}
    >
      {/* Moving "footage" */}
      <div aria-hidden className="absolute inset-0">
        <div className="absolute -left-[10%] top-[-30%] h-[120%] w-[70%] rounded-full bg-[radial-gradient(closest-side,#7c6cff,transparent)] opacity-80 blur-2xl motion-safe:animate-[drift-a_14s_ease-in-out_infinite]" />
        <div className="absolute -right-[5%] bottom-[-35%] h-[120%] w-[65%] rounded-full bg-[radial-gradient(closest-side,#ff5c7f,transparent)] opacity-70 blur-2xl motion-safe:animate-[drift-b_17s_ease-in-out_infinite]" />
        <div className="absolute left-[35%] top-[5%] h-[90%] w-[45%] rounded-full bg-[radial-gradient(closest-side,#22d3ee,transparent)] opacity-55 blur-2xl motion-safe:animate-[drift-a_21s_ease-in-out_infinite_reverse]" />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_50%_120%,rgb(0_0_0/0.65),transparent_60%)]" />
        <div className="absolute inset-y-0 left-0 w-1/4 bg-gradient-to-r from-transparent via-white/10 to-transparent opacity-60 motion-safe:animate-[sweep_7s_ease-in-out_infinite]" />
        <div className="absolute inset-0 opacity-[0.18] [background-image:repeating-linear-gradient(0deg,rgb(255_255_255/0.5)_0_1px,transparent_1px_3px)] mix-blend-overlay" />
      </div>

      {/* Letterbox + frame guides */}
      <div aria-hidden className="absolute inset-x-0 top-0 h-[7%] bg-black/55" />
      <div aria-hidden className="absolute inset-x-0 bottom-0 h-[7%] bg-black/55" />
      <div aria-hidden className="absolute inset-[7%_4%] rounded-lg border border-white/15">
        <span className="absolute -left-px -top-px h-3 w-3 border-l-2 border-t-2 border-white/70" />
        <span className="absolute -right-px -top-px h-3 w-3 border-r-2 border-t-2 border-white/70" />
        <span className="absolute -bottom-px -left-px h-3 w-3 border-b-2 border-l-2 border-white/70" />
        <span className="absolute -bottom-px -right-px h-3 w-3 border-b-2 border-r-2 border-white/70" />
      </div>

      {/* HUD */}
      <div className="absolute inset-x-[6%] top-[10%] flex items-center justify-between font-mono text-[10px] tracking-[0.18em] text-white/80 sm:text-xs">
        <span className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full bg-[#ff4d6d] motion-safe:animate-[blink_1.2s_steps(2)_infinite]" />
          REC
        </span>
        <span className="tabular-nums">{timecode(clock)}</span>
        <span className="hidden sm:inline">9:16 · KARAOKE</span>
      </div>

      {/* Captions */}
      <div className="absolute inset-x-[8%] bottom-[22%] flex flex-wrap items-center justify-center gap-x-[0.35em] gap-y-1 text-center text-[clamp(1.6rem,4.6vw,3.6rem)] font-extrabold leading-[1.05] tracking-[-0.03em] text-white [text-shadow:0_2px_24px_rgb(0_0_0/0.55)]">
        {words.map((text, index) => {
          const active = index === word;
          const past = index < word;
          return (
            <span
              key={`${phrase}-${index}`}
              className={cn(
                'inline-block rounded-xl px-[0.18em] transition-all duration-200',
                active && 'scale-110 bg-white text-[#14102e] shadow-[0_10px_40px_-8px_rgb(124_108_255/0.9)]',
                past && 'text-white',
                !active && !past && 'text-white/25'
              )}
            >
              {text}
            </span>
          );
        })}
      </div>

      {/* Mini timeline */}
      <div className="absolute inset-x-[6%] bottom-[10.5%] flex items-center gap-3">
        <div className="relative h-1.5 flex-1 overflow-hidden rounded-full bg-white/15">
          <div
            className="absolute inset-y-0 left-0 rounded-full bg-white transition-[width] duration-300"
            style={{ width: `${progress * 100}%` }}
          />
        </div>
        <span className="font-mono text-[10px] tracking-widest text-white/70 sm:text-xs">
          {String(Math.max(word, 0) + 1).padStart(2, '0')}/{String(words.length).padStart(2, '0')}
        </span>
      </div>
    </div>
  );
}
