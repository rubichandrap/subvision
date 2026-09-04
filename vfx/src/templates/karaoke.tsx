import React from "react";
import { interpolate, useCurrentFrame, useVideoConfig } from "remotion";

import { SubtitleStyle } from "../contract";
import { ISegment, IWord } from "../types";
import { activeSegment, StyledCaption, TransparentRoot } from "./shared";

// Karaoke captions: only the active segment is on screen, and each word gets
// a highlight swipe that fills left-to-right while the word is spoken. The
// swipe color is the Subtitle Style's highlightColor — the color the old
// template hard-coded, now user-configurable.

export const Karaoke: React.FC<{
  segments: ISegment[];
  style: SubtitleStyle;
}> = ({ segments, style }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const time = frame / fps;

  const segment = activeSegment(segments, time);
  if (!segment) return null;

  const words = segment.words;

  return (
    <TransparentRoot>
      <StyledCaption style={style}>
        <span
          style={{
            display: "inline-flex",
            flexWrap: "wrap",
            justifyContent: "center",
            columnGap: "0.3em",
            rowGap: "0.1em",
          }}
        >
          {words.map((word: IWord, index) => {
            const progress = interpolate(time, [word.start, word.end], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            });
            return (
              <span
                key={index}
                style={{ position: "relative", display: "inline-block" }}
              >
                <span style={{ position: "relative", zIndex: 1 }}>
                  {word.text}
                </span>
                <span
                  style={{
                    position: "absolute",
                    top: 0,
                    left: 0,
                    height: "100%",
                    width: "100%",
                    borderRadius: "0.12em",
                    zIndex: 0,
                    backgroundColor: style.highlightColor,
                    transform: `scaleX(${progress})`,
                    transformOrigin: "left",
                  }}
                />
              </span>
            );
          })}
        </span>
      </StyledCaption>
    </TransparentRoot>
  );
};
