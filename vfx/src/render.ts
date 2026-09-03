import { bundle } from "@remotion/bundler";
import { renderFrames } from "@remotion/renderer";
import os from "os";
import path from "path";

import type { OverlayRenderRequest } from "./services/render-module";

// Renders one job's subtitle overlay frames: bundles the templates and asks
// Remotion for the request's template at the request's frame dimensions with
// the request's (already shifted) segments and Subtitle Style. The frames are
// transparent PNGs the FrameCombiner composites over the video.

const getMaxDurationFrames = (segments: OverlayRenderRequest["segments"], fps: number) => {
  return Math.max(1, Math.ceil(Math.max(...segments.map((s) => s.end)) * fps));
};

export const renderOverlayFrames = async (
  request: OverlayRenderRequest
) => {
  // The bundler needs the templates as TypeScript source (tsc does not copy
  // .tsx into dist), so the entry resolves from the package root, which both
  // `pnpm start` and the container's WORKDIR give us as the cwd.
  const entry = path.join(process.cwd(), "src", "templates", "index.tsx");
  const bundleLocation = await bundle({
    entryPoint: entry,
    outDir: path.join(os.tmpdir(), "remotion-bundle"),
    webpackOverride: (config) => config,
  });

  const durationInFrames = getMaxDurationFrames(request.segments, request.fps);
  await renderFrames({
    serveUrl: bundleLocation,
    composition: {
      defaultCodec: "h264",
      id: request.template,
      width: request.width,
      height: request.height,
      fps: request.fps,
      defaultOutName: request.framesDir,
      defaultProps: {
        segments: [],
        style: request.style,
      },
      props: {
        segments: request.segments,
        style: request.style,
      },
      durationInFrames,
    },
    inputProps: {
      segments: request.segments,
      style: request.style,
    },
    outputDir: request.framesDir,
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
