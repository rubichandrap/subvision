import React from "react";
import { interpolate, useCurrentFrame, useVideoConfig } from "remotion";

import { SubtitleStyle } from "../contract";
import { ISegment } from "../types";
import {
  activeSegment,
  onsetStart,
  StyledCaption,
  TransparentRoot,
} from "./shared";

// Fade captions: the active segment's text eases in, holds, and eases out.

const FADE_SECONDS = 0.4;

export const Fade: React.FC<{
  segments: ISegment[];
  style: SubtitleStyle;
}> = ({ segments, style }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const time = frame / fps;

  const segment = activeSegment(segments, time);
  if (!segment) return null;

  // The fade eases in from the onset, not the segment start: a first word
  // that starts late must not spend the fade invisibly during silence.
  const start = onsetStart(segment);
  const fadeIn = interpolate(
    time,
    [start, start + FADE_SECONDS],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
  );
  const fadeOut = interpolate(
    time,
    [segment.end - FADE_SECONDS, segment.end],
    [1, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
  );
  const opacity = Math.min(fadeIn, fadeOut);

  return (
    <TransparentRoot>
      <StyledCaption style={style}>
        <span style={{ opacity }}>{segment.text}</span>
      </StyledCaption>
    </TransparentRoot>
  );
};
