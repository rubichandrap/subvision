import React from "react";
import { useVideoConfig } from "remotion";

import { SubtitleStyle } from "../contract";
import { ISegment } from "../types";

// Shared rendering for every subtitle template: the Subtitle Style metrics
// and the caption block placement. Templates draw their animation on top of
// <StyledCaption> and render on a transparent root — the overlay frames are
// composited over the video by ffmpeg, so an opaque background here would
// black out the Output.

// The designed caption font size, in px at a 1080-height frame. The style's
// fontSizeScale and the actual composition height scale from here.
export const CAPTION_BASE_FONT_SIZE = 52;

export interface StyleMetrics {
  fontFamily: string;
  fontSize: number;
  bottom: number;
  outlinePx: number;
}

export function useStyleMetrics(style: SubtitleStyle): StyleMetrics {
  const { height } = useVideoConfig();
  const scale = height / 1080;
  return {
    fontFamily: `'${style.fontFamily}', sans-serif`,
    fontSize: Math.max(14, CAPTION_BASE_FONT_SIZE * style.fontSizeScale * scale),
    bottom: style.bottomMargin * height,
    outlinePx: style.outlineWidth * scale,
  };
}

export const TransparentRoot: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => (
  <div style={{ position: "absolute", top: 0, right: 0, bottom: 0, left: 0 }}>
    {children}
  </div>
);

export const StyledCaption: React.FC<{
  style: SubtitleStyle;
  children: React.ReactNode;
}> = ({ style, children }) => {
  const metrics = useStyleMetrics(style);
  return (
    <div
      style={{
        position: "absolute",
        left: 0,
        right: 0,
        bottom: metrics.bottom,
        display: "flex",
        justifyContent: "center",
      }}
    >
      <div
        style={{
          display: "inline-block",
          maxWidth: "86%",
          textAlign: "center",
          fontFamily: metrics.fontFamily,
          fontSize: metrics.fontSize,
          fontWeight: 800,
          lineHeight: 1.25,
          color: style.color,
          textTransform: style.uppercase ? "uppercase" : "none",
          // The outline paints behind the fill, Chromium-side.
          WebkitTextStroke:
            metrics.outlinePx > 0
              ? `${metrics.outlinePx}px ${style.outlineColor}`
              : undefined,
          paintOrder: "stroke fill",
          background:
            style.background === "box"
              ? `rgba(0, 0, 0, ${style.backgroundOpacity})`
              : undefined,
          padding: style.background === "box" ? metrics.fontSize * 0.24 : 0,
          borderRadius: style.background === "box" ? metrics.fontSize * 0.22 : 0,
        }}
      >
        {children}
      </div>
    </div>
  );
};

export interface TimedWord {
  text: string;
  start: number;
  end: number;
}

// Word timings are derived, not transcribed: Transcription Segments carry no
// word-level timestamps, so a segment's duration is distributed across its
// words weighted by their length (+1, so no word is ever instantaneous).
export function wordTimings(segment: ISegment): TimedWord[] {
  const words = segment.text.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return [];
  const weights = words.map((word) => word.length + 1);
  const totalWeight = weights.reduce((sum, weight) => sum + weight, 0);
  const duration = Math.max(0, segment.end - segment.start);
  let cursor = segment.start;
  return words.map((word, index) => {
    const span = (weights[index]! / totalWeight) * duration;
    const timed = { text: word, start: cursor, end: cursor + span };
    cursor += span;
    return timed;
  });
}

// The one segment whose window contains the given time, if any.
export function activeSegment(
  segments: ISegment[],
  time: number
): ISegment | undefined {
  return segments.find(
    (segment) => time >= segment.start && time < segment.end
  );
}
