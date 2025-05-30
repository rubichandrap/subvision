import { bundle } from "@remotion/bundler";
import { renderFrames } from "@remotion/renderer";
import os from "os";
import path from "path";
import { ISegment } from "./types";

const fps = 30;
const getMaxDurationFrames = (segments: ISegment[]) => {
  return Math.ceil(Math.max(...segments.map((s) => s.end)) * fps);
};

export const renderImagesFromTemplate = async (
  segments: ISegment[],
  animationType: string,
  outputDir: string
) => {
  const entry = path.join(__dirname, "templates", "index.tsx");
  const bundleLocation = await bundle({
    entryPoint: entry,
    outDir: path.join(os.tmpdir(), "remotion-bundle"),
    webpackOverride: (config) => config,
  });

  const durationInFrames = getMaxDurationFrames(segments);
  await renderFrames({
    serveUrl: bundleLocation,
    composition: {
      defaultCodec: "h264",
      id: animationType,
      width: 1920,
      height: 1080,
      fps: 30,
      defaultOutName: outputDir,
      defaultProps: {
        segments,
      },
      props: { segments },
      durationInFrames,
    },
    inputProps: { segments },
    outputDir,
    imageFormat: "png",
    frameRange: [0, durationInFrames - 1],
    onFrameUpdate(framesRendered, _, timeToRenderInMilliseconds) {
      console.log(
        `Rendered frame ${framesRendered} of ${durationInFrames} in ${timeToRenderInMilliseconds}ms`
      );
    },
    onStart() {
      console.log(`Started rendering`);
    },
  });
};
