import React from "react";
import { interpolate, useCurrentFrame, useVideoConfig } from "remotion";

import { SubtitleStyle } from "../contract";
import { ISegment } from "../types";
import {
  activeSegment,
  StyledCaption,
  TimedWord,
  TransparentRoot,
  wordTimings,
} from "./shared";

// The shorts-style caption: words pop in one at a time with a springy
// overshoot, the word currently being spoken is highlighted, and words
// already spoken stay on screen dimmed until the segment ends.

// Seconds a word takes to land on its final size.
const POP_IN_SECONDS = 0.16;

// Pop scale curve: start small, overshoot, settle.
function popScale(progress: number): number {
  return interpolate(progress, [0, 0.55, 1], [0.45, 1.14, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
}

function wordOpacity(progress: number): number {
  return interpolate(progress, [0, 0.5], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
}

export const Pop: React.FC<{
  segments: ISegment[];
  style: SubtitleStyle;
}> = ({ segments, style }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const time = frame / fps;

  const segment = activeSegment(segments, time);
  if (!segment) return null;

  const words = wordTimings(segment);
  const lastWordEnd = words.length > 0 ? words[words.length - 1]!.end : segment.end;

  return (
    <TransparentRoot>
      <StyledCaption style={style}>
        <span style={{ display: "inline-flex", flexWrap: "wrap", justifyContent: "center", columnGap: "0.3em", rowGap: "0.1em" }}>
          {words.map((word: TimedWord, index) => {
            if (time < word.start) return null;
            const progress = Math.min(1, (time - word.start) / POP_IN_SECONDS);
            // The last word of a segment stays active until the segment ends.
            const wordEnd = index === words.length - 1 ? lastWordEnd : word.end;
            const isActive = time < wordEnd;
            return (
              <span
                key={index}
                style={{
                  display: "inline-block",
                  opacity: wordOpacity(progress) * (isActive ? 1 : 0.55),
                  transform: `scale(${popScale(progress) * (isActive ? 1.06 : 1)})`,
                  color: isActive ? style.highlightColor : undefined,
                }}
              >
                {word.text}
              </span>
            );
          })}
        </span>
      </StyledCaption>
    </TransparentRoot>
  );
};
