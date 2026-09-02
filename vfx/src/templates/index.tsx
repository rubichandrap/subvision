import React from "react";
import { Composition, registerRoot } from "remotion";

import { DEFAULT_STYLE } from "../contract";
import "../styles/tailwind.css";

// Fonts bundled with the service (see the FONT_FAMILIES contract): a caption
// rendered server-side can only use faces that ship with the image. The
// default import is the regular weight; captions use the bold weights too.
import "@fontsource/montserrat/400.css";
import "@fontsource/montserrat/700.css";
import "@fontsource/montserrat/800.css";
import "@fontsource/inter/400.css";
import "@fontsource/inter/700.css";
import "@fontsource/inter/900.css";
import "@fontsource/poppins/400.css";
import "@fontsource/poppins/600.css";
import "@fontsource/poppins/700.css";
import "@fontsource/oswald/400.css";
import "@fontsource/oswald/500.css";
import "@fontsource/oswald/700.css";
import "@fontsource/bebas-neue/400.css";
import "@fontsource/anton/400.css";

import { Fade } from "./fade";
import { Karaoke } from "./karaoke";
import { Pop } from "./pop";
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
        defaultProps={{ segments: [], style: DEFAULT_STYLE }}
      />
      <Composition
        id="slide"
        component={Slide}
        durationInFrames={150}
        fps={30}
        width={1920}
        height={1080}
        defaultProps={{ segments: [], style: DEFAULT_STYLE }}
      />
      <Composition
        id="karaoke"
        component={Karaoke}
        durationInFrames={150}
        fps={30}
        width={1080}
        height={1920}
        defaultProps={{ segments: [], style: DEFAULT_STYLE }}
      />
      <Composition
        id="pop"
        component={Pop}
        durationInFrames={150}
        fps={30}
        width={1080}
        height={1920}
        defaultProps={{ segments: [], style: DEFAULT_STYLE }}
      />
    </>
  );
};

registerRoot(RemotionRoot);

export default RemotionRoot;
