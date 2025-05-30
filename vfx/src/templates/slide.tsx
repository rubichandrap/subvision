import { ISegment } from "@/types";
import { interpolate, useCurrentFrame } from "remotion";

const fps = 30;

export const Slide: React.FC<{
  segments: ISegment[];
}> = ({ segments }) => {
  const frame = useCurrentFrame();
  const frameTime = frame / fps;

  const activeSegment = segments.find(
    (s) => frameTime >= s.start && frameTime <= s.end
  );

  if (!activeSegment) return null;

  const progress = interpolate(
    frameTime,
    [activeSegment.start, activeSegment.end],
    [0, 1],
    {
      extrapolateLeft: "clamp",
      extrapolateRight: "clamp",
    }
  );

  const translateX = interpolate(progress, [0, 1], [-400, 0]);

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
      <h1 style={{ transform: `translateX(${translateX}px)`, fontSize: 48 }}>
        {activeSegment.text}
      </h1>
    </div>
  );
};
