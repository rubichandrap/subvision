import React from "react";
import { Composition, registerRoot } from "remotion";
import { Fade } from "./fade";
import { Karaoke } from "./karaoke";
import { Slide } from "./slide";

const RemotionRoot: React.FC = () => {
  return (
    <>
      <Composition
        id="fade"
        component={Fade}
        durationInFrames={150}
        fps={30}
        width={1920}
        height={1080}
        defaultProps={{ segments: [] }}
      />
      <Composition
        id="slide"
        component={Slide}
        durationInFrames={150}
        fps={30}
        width={1920}
        height={1080}
        defaultProps={{ segments: [] }}
      />
      <Composition
        id="karaoke"
        component={Karaoke}
        durationInFrames={150}
        fps={30}
        width={1080}
        height={1920}
        defaultProps={{ segments: [] }}
      />
    </>
  );
};

registerRoot(RemotionRoot);

export default RemotionRoot;
