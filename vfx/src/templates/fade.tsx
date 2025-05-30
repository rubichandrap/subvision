import { ISegment } from "@/types";
import { interpolate, useCurrentFrame } from "remotion";

const fps = 30;

export const Fade: React.FC<{
  segments: ISegment[];
}> = ({ segments }) => {
  const frame = useCurrentFrame();
  const frameTime = frame / fps;

  const activeSegment = segments.find(
    (s) => frameTime >= s.start && frameTime <= s.end
  );

  if (!activeSegment) return null;

  const fadeIn = interpolate(
    frameTime,
    [activeSegment.start, activeSegment.start + 0.5],
    [0, 1],
    { extrapolateRight: "clamp" }
  );
  const fadeOut = interpolate(
    frameTime,
    [activeSegment.end - 0.5, activeSegment.end],
    [1, 0],
    { extrapolateLeft: "clamp" }
  );
  const opacity = Math.min(fadeIn, fadeOut);

  return (
    <div
      style={{
        flex: 1,
        color: "white",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      <h1 style={{ fontSize: 50, opacity }}>{activeSegment.text}</h1>
    </div>
  );
};
