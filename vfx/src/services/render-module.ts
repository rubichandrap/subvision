import path from "path";

import { outputKey, uploadKey } from "../config/storage";
import { VfxJobPayload } from "../contract";
import { ensureDirs } from "../utils/ensure-dirs";

// The render module owns everything render-related: the id derived from the
// object key, every temporary path (including file extensions), the render
// options, the ffmpeg invocation, and the Output upload. Callers pass a job
// and receive an outcome; they make no path or option decisions.

export interface RenderOptions {
  fps: number;
  width: number;
  height: number;
}

export interface ObjectStorage {
  downloadFile(bucketName: string, objectKey: string, filePath: string): Promise<string>;
  uploadFile(bucketName: string, objectKey: string, filePath: string): Promise<void>;
}

// Renders the subtitle frames for the segments into framesDir as
// element-%03d.png; the Remotion-backed implementation is wired in index.ts.
export interface FrameRenderer {
  (segments: VfxJobPayload["segments"], framesDir: string): Promise<void>;
}

// Overlays the rendered frames onto the video and writes outputPath.
export interface FrameCombiner {
  (videoPath: string, framesDir: string, outputPath: string, fps: number): Promise<void>;
}

export interface RenderModuleConfig {
  tmpDir: string;
  bucket: string;
  options: RenderOptions;
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
    private readonly config: RenderModuleConfig
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

    await this.renderFrames(job.segments, framesDir);
    await this.combineFrames(videoPath, framesDir, outputPath, this.config.options.fps);

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
