'use client';

import * as React from 'react';

import { clamp, type FrameState, type SubtitleStyleState } from '@/lib/edit-spec';
import { captionFontFamily } from '@/lib/caption-fonts';

// The editor stage: the video reframed live into the target Frame. The video
// element is scaled to cover the frame box (object-cover semantics computed
// by hand so zoom and pan compose), panned by dragging, and the sample
// caption is styled exactly the way the vfx service will burn it —
// proportions are derived from the frame height just like the server side
// (52px reference font at a 1080-height frame).

const CAPTION_BASE_FONT_SIZE = 52;
const SAMPLE_CAPTION = 'your captions will look like this';
const FREE_RATIO_MIN = 0.5;
const FREE_RATIO_MAX = 2;

export interface FramePreviewProps {
  src: string;
  frame: FrameState;
  style: SubtitleStyleState;
  animation: string;
  videoRef: React.RefObject<HTMLVideoElement | null>;
  onLoadedMetadata: (event: React.SyntheticEvent<HTMLVideoElement>) => void;
  onTimeUpdate: (event: React.SyntheticEvent<HTMLVideoElement>) => void;
  onPan: (panX: number, panY: number) => void;
  onFreeRatio: (ratio: number) => void;
  disabled?: boolean;
}

export function FramePreview({
  src,
  frame,
  style,
  animation,
  videoRef,
  onLoadedMetadata,
  onTimeUpdate,
  onPan,
  onFreeRatio,
  disabled,
}: FramePreviewProps) {
  const availableRef = React.useRef<HTMLDivElement>(null);
  const [available, setAvailable] = React.useState({ width: 0, height: 0 });
  const [videoSize, setVideoSize] = React.useState({ width: 0, height: 0 });
  const [panning, setPanning] = React.useState(false);

  React.useEffect(() => {
    const element = availableRef.current;
    if (!element) return;
    const observer = new ResizeObserver(([entry]) => {
      if (!entry) return;
      setAvailable({
        width: entry.contentRect.width,
        height: entry.contentRect.height,
      });
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  // Fit the target ratio into the available stage area.
  const ratio = frame.ratio;
  const fits =
    available.width > 0 && available.height > 0 && Number.isFinite(ratio) && ratio > 0;
  const boxSize = fits
    ? available.width / available.height > ratio
      ? { width: available.height * ratio, height: available.height }
      : { width: available.width, height: available.width / ratio }
    : { width: 0, height: 0 };

  const coverScale =
    videoSize.width > 0 && boxSize.width > 0
      ? Math.max(boxSize.width / videoSize.width, boxSize.height / videoSize.height)
      : 1;
  const totalScale = coverScale * frame.zoom;
  const overflowX = Math.max(0, videoSize.width * totalScale - boxSize.width) / 2;
  const overflowY = Math.max(0, videoSize.height * totalScale - boxSize.height) / 2;
  const panX = frame.panX * overflowX;
  const panY = frame.panY * overflowY;

  const handlePanPointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    if (disabled || (overflowX === 0 && overflowY === 0)) return;
    event.preventDefault();
    setPanning(true);
    const startX = event.clientX;
    const startY = event.clientY;
    const startPanX = frame.panX;
    const startPanY = frame.panY;

    const onMove = (moveEvent: PointerEvent) => {
      const nextPanX =
        overflowX > 0
          ? clamp(startPanX + (moveEvent.clientX - startX) / overflowX, -1, 1)
          : startPanX;
      const nextPanY =
        overflowY > 0
          ? clamp(startPanY + (moveEvent.clientY - startY) / overflowY, -1, 1)
          : startPanY;
      onPan(nextPanX, nextPanY);
    };
    const onUp = () => {
      setPanning(false);
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp);
  };

  const handleFreeRatioPointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    if (disabled) return;
    event.preventDefault();
    event.stopPropagation();
    const rect = event.currentTarget.parentElement?.getBoundingClientRect();
    if (!rect) return;

    const onMove = (moveEvent: PointerEvent) => {
      const width = Math.max(1, moveEvent.clientX - rect.left);
      const height = Math.max(1, moveEvent.clientY - rect.top);
      onFreeRatio(clamp(width / height, FREE_RATIO_MIN, FREE_RATIO_MAX));
    };
    const onUp = () => {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp);
  };

  // Caption metrics: same proportion math as the vfx templates.
  const captionScale = boxSize.height > 0 ? boxSize.height / 1080 : 1;
  const captionFontSize = Math.max(
    10,
    CAPTION_BASE_FONT_SIZE * style.fontSizeScale * captionScale
  );
  const words = SAMPLE_CAPTION.split(' ');
  const emphasisIndex = 2;
  const highlightWord = animation === 'karaoke' || animation === 'pop';

  return (
    <div
      ref={availableRef}
      className="flex min-h-0 w-full flex-1 self-stretch items-center justify-center"
    >
      <div
        className="relative overflow-hidden rounded-xl border border-border/80 bg-black shadow-2xl"
        style={{
          width: boxSize.width || undefined,
          height: boxSize.height || undefined,
        }}
      >
        <div
          onPointerDown={handlePanPointerDown}
          className={`absolute inset-0 select-none ${
            panning
              ? 'cursor-grabbing'
              : overflowX > 0 || overflowY > 0
                ? 'cursor-grab'
                : 'cursor-default'
          }`}
        >
          <video
            ref={videoRef}
            src={src}
            muted
            playsInline
            className="absolute left-1/2 top-1/2 max-w-none"
            style={{
              transform: `translate(-50%, -50%) translate(${panX}px, ${panY}px) scale(${totalScale})`,
              transformOrigin: 'center',
            }}
            onLoadedMetadata={(event) => {
              const video = event.currentTarget;
              setVideoSize({ width: video.videoWidth, height: video.videoHeight });
              onLoadedMetadata(event);
            }}
            onTimeUpdate={onTimeUpdate}
          />
          {/* Sample caption, styled like the render */}
          <div
            className="pointer-events-none absolute left-0 right-0 flex justify-center px-[7%]"
            style={{ bottom: style.bottomMargin * boxSize.height }}
          >
            <span
              style={{
                display: 'inline-block',
                maxWidth: '100%',
                textAlign: 'center',
                fontFamily: captionFontFamily(style.fontFamily),
                fontSize: captionFontSize,
                fontWeight: 800,
                lineHeight: 1.25,
                color: style.color,
                textTransform: style.uppercase ? 'uppercase' : 'none',
                WebkitTextStroke:
                  style.outlineWidth > 0
                    ? `${style.outlineWidth * captionScale}px ${style.outlineColor}`
                    : undefined,
                paintOrder: 'stroke fill',
                background:
                  style.background === 'box'
                    ? `rgba(0, 0, 0, ${style.backgroundOpacity})`
                    : undefined,
                padding: style.background === 'box' ? captionFontSize * 0.24 : 0,
                borderRadius: style.background === 'box' ? captionFontSize * 0.22 : 0,
              }}
            >
              {words.map((word, index) => {
                if (!highlightWord || index !== emphasisIndex) {
                  return <React.Fragment key={index}>{word} </React.Fragment>;
                }
                return animation === 'karaoke' ? (
                  <span
                    key={index}
                    style={{
                      backgroundColor: style.highlightColor,
                      padding: '0 0.08em',
                      borderRadius: '0.08em',
                    }}
                  >
                    {word}{' '}
                  </span>
                ) : (
                  <span key={index} style={{ color: style.highlightColor }}>
                    {word}{' '}
                  </span>
                );
              })}
            </span>
          </div>
        </div>

        {frame.preset === 'free' && !disabled && (
          <div
            role="slider"
            aria-label="Drag to resize the frame"
            aria-valuenow={Number(frame.ratio.toFixed(2))}
            aria-valuemin={FREE_RATIO_MIN}
            aria-valuemax={FREE_RATIO_MAX}
            tabIndex={0}
            onPointerDown={handleFreeRatioPointerDown}
            onKeyDown={(event) => {
              const step = 0.05;
              if (event.key === 'ArrowLeft' || event.key === 'ArrowDown') {
                event.preventDefault();
                onFreeRatio(clamp(frame.ratio - step, FREE_RATIO_MIN, FREE_RATIO_MAX));
              }
              if (event.key === 'ArrowRight' || event.key === 'ArrowUp') {
                event.preventDefault();
                onFreeRatio(clamp(frame.ratio + step, FREE_RATIO_MIN, FREE_RATIO_MAX));
              }
            }}
            className="absolute bottom-2 right-2 flex h-8 w-8 cursor-nwse-resize items-center justify-center rounded-full border border-white/40 bg-black/60 text-white backdrop-blur transition-colors hover:bg-black/80"
          >
            <svg
              viewBox="0 0 16 16"
              className="h-4 w-4"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.6"
            >
              <path
                d="M10 2h4v4M6 14H2v-4M14 2 9 7M2 14l5-5"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </div>
        )}
      </div>
    </div>
  );
}
