import { spawn } from "child_process";
import fs from "fs";
import path from "path";

import { outputKey, uploadKey } from "../config/storage";
import { DEFAULT_STYLE, DEFAULT_WORDS_PER_PAGE, EditSpec, Frame, SubtitleStyle, VfxJobPayload } from "../contract";
import { ISegment, IWord } from "../types";
import { ensureDirs } from "../utils/ensure-dirs";

// The render module owns everything render-related: the id derived from the
// object key, every temporary path (including file extensions), the render
// options, the Edit Spec application (trim, crop-to-fill, animation, style),
// the ffmpeg invocation, and the Output upload. Callers pass a job and
// receive an outcome; they make no path or option decisions.

export interface RenderOptions {
  fps: number;
  width: number;
  height: number;
}

export interface ObjectStorage {
  downloadFile(bucketName: string, objectKey: string, filePath: string): Promise<string>;
  uploadFile(bucketName: string, objectKey: string, filePath: string): Promise<void>;
}

// What the subtitle overlay renderer receives for one job: the segments
// already shifted into the trim window, the frame's pixel dimensions, the
// animation template to render, the Subtitle Style to style it with, and the
// Caption Page size to page pop/karaoke with.
export interface OverlayRenderRequest {
  segments: ISegment[];
  framesDir: string;
  width: number;
  height: number;
  fps: number;
  template: string;
  style: SubtitleStyle;
  wordsPerPage: number;
}

// Renders the subtitle frames for the request into framesDir as
// element-%03d.png; the Remotion-backed implementation is wired in index.ts.
export interface FrameRenderer {
  (request: OverlayRenderRequest): Promise<void>;
}

// How the FrameCombiner cuts and reframes the video before the overlay lands
// on it, derived from the Edit Spec.
export interface CombinePlan {
  /** ffmpeg filters applied to the video input (scale, crop) before the overlay. */
  videoFilters: string[];
  /** Where the rendered window starts in the source video, seconds. */
  seekStart: number;
  /** Window length, seconds; null renders to the end of the video. */
  seekDuration: number | null;
  /** Seconds into the output after which the subtitle overlay is disabled. */
  overlayUntil: number | null;
}

// Overlays the rendered frames onto the video and writes outputPath.
export interface FrameCombiner {
  (videoPath: string, framesDir: string, outputPath: string, fps: number, plan: CombinePlan): Promise<void>;
}

export interface VideoProbe {
  width: number;
  height: number;
  duration: number | null;
}

// Reads the video's dimensions; the ffprobe-backed implementation is wired
// in New (via probeVideoFile). Injectable so tests can fake it.
export interface VideoProber {
  (videoPath: string): Promise<VideoProbe>;
}

export interface RenderModuleConfig {
  tmpDir: string;
  bucket: string;
  options: RenderOptions;
  // The composition id of the fallback subtitle template for jobs that carry
  // no Edit Spec (RENDER_TEMPLATE).
  template: string;
}

export interface RenderOutcome {
  uploadId: string;
  outputKey: string;
}

export class RenderModule {
  constructor(
    private readonly storage: ObjectStorage,
    private readonly renderFrames: FrameRenderer,
    private readonly combineFrames: FrameCombiner,
    private readonly config: RenderModuleConfig,
    private readonly probe: VideoProber = probeVideoFile
  ) {}

  // run renders one VFX Job: it throws if any stage fails and resolves with
  // the Output's location once it is stored at outputs/<id>.
  async run(job: VfxJobPayload): Promise<RenderOutcome> {
    const id = jobIdFromObjectKey(job.objectKey);
    if (job.segments.length === 0) {
      throw new Error(`job for upload ${job.uploadId} carries no segments to render`);
    }

    const videoPath = this.videoPath(id);
    const framesDir = this.framesDir(id);
    const outputPath = this.outputPath(id);

    ensureDirs(path.dirname(videoPath), framesDir, path.dirname(outputPath));

    await this.storage.downloadFile(this.config.bucket, uploadKey(id), videoPath);

    if (!job.editSpec) {
      // No Edit Spec: render the whole video at the configured dimensions
      // with the fallback template and the default style, exactly as the
      // pipeline did before the editor existed.
      await this.renderFrames({
        segments: job.segments,
        framesDir,
        width: this.config.options.width,
        height: this.config.options.height,
        fps: this.config.options.fps,
        template: this.config.template,
        style: DEFAULT_STYLE,
        wordsPerPage: DEFAULT_WORDS_PER_PAGE,
      });
      await this.combineFrames(videoPath, framesDir, outputPath, this.config.options.fps, {
        videoFilters: [],
        seekStart: 0,
        seekDuration: null,
        overlayUntil: maxSegmentEnd(job.segments),
      });
      await this.storage.uploadFile(this.config.bucket, outputKey(id), outputPath);
      return { uploadId: job.uploadId, outputKey: outputKey(id) };
    }

    // With an Edit Spec the source video is probed, reframed to the target
    // Frame (crop-to-fill), and cut to the trim window in the same pass; the
    // subtitle overlay is rendered at the frame's dimensions for the shifted
    // segments. The Output is the first and only encode.
    const probe = await this.probe(videoPath);
    const framePlan = planFrame(job.editSpec.frame, probe.width, probe.height);
    const segments = shiftSegments(job.segments, job.editSpec.trim);

    await this.renderFrames({
      segments,
      framesDir,
      width: framePlan.width,
      height: framePlan.height,
      fps: this.config.options.fps,
      template: job.editSpec.animation,
      style: job.editSpec.style,
      wordsPerPage: job.editSpec.captions.wordsPerPage,
    });
    await this.combineFrames(videoPath, framesDir, outputPath, this.config.options.fps, {
      videoFilters: framePlan.filters,
      seekStart: job.editSpec.trim.start,
      seekDuration:
        job.editSpec.trim.end > 0 ? job.editSpec.trim.end - job.editSpec.trim.start : null,
      overlayUntil: maxSegmentEnd(segments),
    });

    await this.storage.uploadFile(this.config.bucket, outputKey(id), outputPath);
    return { uploadId: job.uploadId, outputKey: outputKey(id) };
  }

  private videoPath(id: string): string {
    // The upload's container is probed by ffmpeg from content, so the
    // downloaded video needs no extension; the rendered Output does.
    return path.join(this.config.tmpDir, "videos", id);
  }

  private framesDir(id: string): string {
    return path.join(this.config.tmpDir, "frames", id);
  }

  private outputPath(id: string): string {
    return path.join(this.config.tmpDir, "outputs", `${id}.mp4`);
  }
}

// jobIdFromObjectKey derives the id from an object key: the last path segment.
export function jobIdFromObjectKey(objectKey: string): string {
  const id = objectKey.split("/").pop();
  if (!id) {
    throw new Error(`object key ${JSON.stringify(objectKey)} carries no id`);
  }
  return id;
}

// maxSegmentEnd is how long the subtitle overlay must stay enabled: until the
// last segment ends. null when there is nothing to show at all.
export function maxSegmentEnd(segments: ISegment[]): number | null {
  if (segments.length === 0) return null;
  return Math.max(...segments.map((segment) => segment.end));
}

// shiftSegments translates absolute Transcription Segment times into the
// trim window's local times and drops segments outside it. Segments ending
// exactly at the window start (or starting exactly at a finite window end)
// contribute nothing. Their Timed Words follow the same translation and
// clipping: words entirely outside the window contribute nothing, the rest
// clamp into the window edges.
export function shiftSegments(
  segments: ISegment[],
  trim: EditSpec["trim"]
): ISegment[] {
  const windowEnd = trim.end > 0 ? trim.end : Infinity;
  return segments
    .filter((segment) => segment.end > trim.start && segment.start < windowEnd)
    .map((segment) => ({
      ...segment,
      start: Math.max(0, segment.start - trim.start),
      end: Math.min(segment.end, windowEnd) - trim.start,
      words: shiftWords(segment.words, trim, windowEnd),
    }));
}

function shiftWords(words: IWord[], trim: EditSpec["trim"], windowEnd: number): IWord[] {
  return words
    .filter((word) => word.end > trim.start && word.start < windowEnd)
    .map((word) => ({
      text: word.text,
      start: Math.max(0, word.start - trim.start),
      end: Math.min(word.end, windowEnd) - trim.start,
    }));
}

// evenDown rounds a dimension down to the nearest even number: h264 demands
// even sizes.
function evenDown(value: number): number {
  return Math.max(2, Math.floor(value / 2) * 2);
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

// planFrame reframes the source video into the Edit Spec's Frame: the target
// dimensions and the ffmpeg scale→crop filters implementing crop-to-fill with
// the requested zoom and pan. The target's long side aims at 1920 but the
// source is never upscaled past what covering it requires at zoom 1; pan is
// normalized (-1..1) across the overflow: -1 pins the crop window to the
// left/top edge, 1 to the right/bottom.
export function planFrame(
  frame: Frame,
  sourceWidth: number,
  sourceHeight: number
): { width: number; height: number; filters: string[] } {
  if (sourceWidth <= 0 || sourceHeight <= 0) {
    throw new Error(`probed video carries invalid dimensions ${sourceWidth}x${sourceHeight}`);
  }

  const desiredLongSide = 1920;
  let targetWidth =
    frame.ratio >= 1 ? desiredLongSide : desiredLongSide * frame.ratio;
  let targetHeight =
    frame.ratio >= 1 ? desiredLongSide / frame.ratio : desiredLongSide;
  const coverBase = Math.max(targetWidth / sourceWidth, targetHeight / sourceHeight);
  if (coverBase > 1) {
    // Upscaling would soften the source; shrink the frame to the source instead.
    targetWidth /= coverBase;
    targetHeight /= coverBase;
  }
  const width = evenDown(targetWidth);
  const height = evenDown(width / frame.ratio); // keep the ratio exact through rounding

  const coverScale = Math.max(width / sourceWidth, height / sourceHeight) * frame.zoom;
  const scaledWidth = Math.round(sourceWidth * coverScale);
  const scaledHeight = Math.round(sourceHeight * coverScale);

  const maxX = Math.max(0, scaledWidth - width);
  const maxY = Math.max(0, scaledHeight - height);
  const x = clamp(Math.round(((scaledWidth - width) / 2) * (1 + frame.panX)), 0, maxX);
  const y = clamp(Math.round(((scaledHeight - height) / 2) * (1 + frame.panY)), 0, maxY);

  return {
    width,
    height,
    filters: [
      `scale=${scaledWidth}:${scaledHeight}`,
      `crop=${width}:${height}:${x}:${y}`,
    ],
  };
}

// probeVideoFile reads the video's dimensions and duration with ffprobe.
export async function probeVideoFile(videoPath: string): Promise<VideoProbe> {
  const ffprobe = spawn("ffprobe", [
    "-v",
    "error",
    "-select_streams",
    "v:0",
    "-show_entries",
    "stream=width,height:format=duration",
    "-of",
    "json",
    videoPath,
  ]);

  let stdout = "";
  let stderr = "";
  ffprobe.stdout.on("data", (data) => {
    stdout += data;
  });
  ffprobe.stderr.on("data", (data) => {
    stderr += data;
  });

  const code = await new Promise<number>((resolve, reject) => {
    ffprobe.on("error", reject);
    ffprobe.on("close", resolve);
  });
  if (code !== 0) {
    throw new Error(`ffprobe failed for ${videoPath} with code ${code}: ${stderr.trim()}`);
  }

  let parsed: {
    streams?: Array<{ width?: number; height?: number }>;
    format?: { duration?: string };
  };
  try {
    parsed = JSON.parse(stdout);
  } catch (error) {
    throw new Error(`ffprobe produced unparseable output for ${videoPath}: ${error}`);
  }
  const stream = parsed.streams?.[0];
  if (!stream || typeof stream.width !== "number" || typeof stream.height !== "number") {
    throw new Error(`ffprobe found no video stream with dimensions in ${videoPath}`);
  }
  const duration = parsed.format?.duration ? Number(parsed.format.duration) : null;
  return {
    width: stream.width,
    height: stream.height,
    duration: duration !== null && Number.isFinite(duration) ? duration : null,
  };
}

// combineFramesWithFFmpeg is the module's default FrameCombiner: it cuts the
// trim window, applies the plan's video filters, overlays the rendered
// frames, and encodes the Output with ffmpeg. The overlay is disabled once
// the plan's overlay window has passed, so the video plays to its (trimmed)
// end with no frozen caption; the frames input itself only reaches that far.
// overlaySequencePattern infers the image2 printf pattern from the frames
// Remotion actually wrote: Remotion pads filenames to the width of the
// largest frame index (String(lastFrame).length), so the pattern depends on
// the render's duration — %03d under 1000 frames, %04d above, and so on.
// Reading the longest filename back means the combiner can never disagree
// with the renderer about padding again.
export async function overlaySequencePattern(framesDir: string): Promise<string | null> {
  const entries = await fs.promises.readdir(framesDir);
  const frames = entries.filter((name) => /^element-\d+\.png$/.test(name));
  if (frames.length === 0) return null;
  const widest = frames.reduce((acc, name) => Math.max(acc, name.length), 0);
  // "element-0000.png" is 16 chars: prefix+dash+pad+digits… pad = widest - 9
  const pad = widest - "element-".length - ".png".length;
  return `${framesDir}/element-%0${pad}d.png`;
}

export async function combineFramesWithFFmpeg(
  videoPath: string,
  framesDir: string,
  outputPath: string,
  fps: number,
  plan: CombinePlan
): Promise<void> {
  const sequence = await overlaySequencePattern(framesDir);
  if (sequence === null) {
    throw new Error(
      `no overlay frames found in ${framesDir}; the frame renderer wrote nothing to combine`
    );
  }
  const overlay = `[0:v]overlay=0:0${
    plan.overlayUntil !== null
      ? `:enable='lte(t,${Number(plan.overlayUntil.toFixed(3))})'`
      : ""
  }[v]`;
  // With filters the video lands on an intermediate label the overlay reads
  // from; without them the two inputs feed the overlay directly.
  const filterGraph =
    plan.videoFilters.length > 0
      ? `[1:v]${plan.videoFilters.join(",")}[bg];[bg]${overlay}`
      : `[1:v]${overlay}`;

  const args = [
    "-framerate",
    String(fps),
    "-i",
    sequence, // overlay (padding inferred from the frames actually written)
    "-ss",
    String(plan.seekStart),
    "-i",
    videoPath, // background
  ];
  if (plan.seekDuration !== null) {
    args.push("-t", String(plan.seekDuration));
  }
  args.push(
    "-filter_complex",
    filterGraph,
    "-map",
    "[v]",
    "-map",
    "1:a?",
    "-c:v",
    "libx264",
    "-crf",
    "23",
    "-preset",
    "fast",
    "-c:a",
    "aac",
    outputPath
  );

  const ffmpeg = spawn("ffmpeg", args);

  // Log FFmpeg output for debugging
  ffmpeg.stdout.on("data", (data) => {
    console.log(`FFmpeg Output: ${data}`);
  });

  ffmpeg.stderr.on("data", (data) => {
    console.error(`FFmpeg Error: ${data}`);
  });

  // Handle FFmpeg process completion
  return new Promise((resolve, reject) => {
    ffmpeg.on("close", (code) => {
      if (code === 0) {
        console.log(`Video created successfully: ${outputPath}`);
        resolve();
      } else {
        reject(new Error(`FFmpeg process failed with code ${code}`));
      }
    });
  });
}
