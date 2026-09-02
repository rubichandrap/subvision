import React from "react";
import { interpolate, useCurrentFrame, useVideoConfig } from "remotion";

import { SubtitleStyle } from "../contract";
import { ISegment } from "../types";
import {
  activeSegment,
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

  const fadeIn = interpolate(
    time,
    [segment.start, segment.start + FADE_SECONDS],
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
