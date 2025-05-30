import React from "react";
import { interpolate, useCurrentFrame, useVideoConfig } from "remotion";

const dummyWords = [
  { word: "Hello", start: 0, end: 0.5 },
  { word: "world", start: 0.5, end: 1.0 },
  { word: "this", start: 1.0, end: 1.3 },
  { word: "is", start: 1.3, end: 1.5 },
  { word: "karaoke!", start: 1.5, end: 2.0 },
];

export const Karaoke: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  return (
    <div className="flex flex-col items-center justify-center w-full h-full bg-black text-white text-6xl font-bold">
      <div className="flex flex-wrap gap-4">
        {dummyWords.map(({ word, start, end }, i) => {
          const startFrame = start * fps;
          const endFrame = end * fps;
          const progress = interpolate(
            frame,
            [startFrame, endFrame],
            [0, 1],
            { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
          );

          return (
            <div key={i} className="relative px-2">
              <span className="relative z-10">{word}</span>
              <div
                className="absolute top-0 left-0 h-full w-full rounded-md z-0"
                style={{
                  backgroundColor: "#facc15", // yellow-400
                  transform: `scaleX(${progress})`,
                  transformOrigin: "left",
                  transition: "transform 0.1s",
                }}
              ></div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
