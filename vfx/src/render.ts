import { bundle } from "@remotion/bundler";
import { renderFrames } from "@remotion/renderer";
import os from "os";
import path from "path";
import { ISegment } from "./types";

export interface RenderFrameOptions {
  fps: number;
  width: number;
  height: number;
}

const getMaxDurationFrames = (segments: ISegment[], fps: number) => {
  return Math.ceil(Math.max(...segments.map((s) => s.end)) * fps);
};

export const renderImagesFromTemplate = async (
  segments: ISegment[],
  template: string,
  outputDir: string,
  options: RenderFrameOptions
) => {
  const entry = path.join(__dirname, "templates", "index.tsx");
  const bundleLocation = await bundle({
    entryPoint: entry,
    outDir: path.join(os.tmpdir(), "remotion-bundle"),
    webpackOverride: (config) => config,
  });

  const durationInFrames = getMaxDurationFrames(segments, options.fps);
  await renderFrames({
    serveUrl: bundleLocation,
    composition: {
      defaultCodec: "h264",
      id: template,
      width: options.width,
      height: options.height,
      fps: options.fps,
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
