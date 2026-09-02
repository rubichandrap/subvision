import React from "react";
import { interpolate, useCurrentFrame, useVideoConfig } from "remotion";

import { SubtitleStyle } from "../contract";
import { ISegment } from "../types";
import {
  activeSegment,
  StyledCaption,
  TransparentRoot,
} from "./shared";

// Slide captions: the active segment's text slides in from the left and
// settles, then leaves with the segment.

const SLIDE_IN_SECONDS = 0.35;

export const Slide: React.FC<{
  segments: ISegment[];
  style: SubtitleStyle;
}> = ({ segments, style }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const time = frame / fps;

  const segment = activeSegment(segments, time);
  if (!segment) return null;

  const slideIn = interpolate(
    time,
    [segment.start, segment.start + SLIDE_IN_SECONDS],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
  );
  const eased = 1 - Math.pow(1 - slideIn, 3);

  return (
    <TransparentRoot>
      <StyledCaption style={style}>
        <span
          style={{
            display: "inline-block",
            opacity: eased,
            transform: `translateX(${(eased - 1) * 0.6}em)`,
          }}
        >
          {segment.text}
        </span>
      </StyledCaption>
    </TransparentRoot>
  );
};
