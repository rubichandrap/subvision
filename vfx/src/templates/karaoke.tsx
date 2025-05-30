import { ISegment } from "@/types";
import React from "react";
import { interpolate, useCurrentFrame, useVideoConfig } from "remotion";

export const Karaoke: React.FC<{
  segments: ISegment[];
}> = ({ segments }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  return (
    <div className="flex flex-col items-center justify-center w-full h-full bg-black text-white text-6xl font-bold">
      <div className="flex flex-wrap gap-4">
        {segments.map(({ text, start, end }, i) => {
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
              <span className="relative z-10">{text}</span>
              <div
                className="absolute top-0 left-0 h-full w-full rounded-md z-0"
                style={{
                  backgroundColor: "#facc15", // TODO: this color should came from user request (message queue)
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
