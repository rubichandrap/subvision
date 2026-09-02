import assert from "node:assert/strict";
import fs from "fs";
import os from "os";
import path from "path";
import { describe, it } from "node:test";

import { RenderModule, RenderModuleConfig } from "./render-module";
import { ISegment } from "../types";

const segments: ISegment[] = [
  { start: 0, end: 1.5, text: "hello" },
  { start: 1.5, end: 3, text: "world" },
];

interface StorageCall {
  bucket: string;
  key: string;
  filePath: string;
}

class FakeStorage {
  downloads: StorageCall[] = [];
  uploads: StorageCall[] = [];

  async downloadFile(bucket: string, key: string, filePath: string): Promise<string> {
    this.downloads.push({ bucket, key, filePath });
    return filePath;
  }

  async uploadFile(bucket: string, key: string, filePath: string): Promise<void> {
    this.uploads.push({ bucket, key, filePath });
  }
}

function makeModule(tmpDir: string, storage: FakeStorage, fps = 30) {
  const config: RenderModuleConfig = {
    tmpDir,
    bucket: "subvision",
    options: { fps, width: 1920, height: 1080 },
  };
  const renderFramesCalls: Array<{ segments: ISegment[]; framesDir: string }> = [];
  const combineCalls: Array<{
    videoPath: string;
    framesDir: string;
    outputPath: string;
    fps: number;
  }> = [];
  const renderFrames = async (segs: ISegment[], framesDir: string) => {
    renderFramesCalls.push({ segments: segs, framesDir });
  };
  const combineFrames = async (
    videoPath: string,
    framesDir: string,
    outputPath: string,
    frameRate: number
  ) => {
    combineCalls.push({ videoPath, framesDir, outputPath, fps: frameRate });
  };
  const module = new RenderModule(storage, renderFrames, combineFrames, config);
  return { module, renderFramesCalls, combineCalls };
}

describe("render module", () => {
  it("a synthetic VFX Job results in an Output object at outputs/<id>", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const storage = new FakeStorage();
    const { module, combineCalls } = makeModule(tmpDir, storage);

    const outcome = await module.run({
      uploadId: "u1",
      objectKey: "uploads/u1",
      segments,
    });

    assert.deepEqual(outcome, { uploadId: "u1", outputKey: "outputs/u1" });
    assert.equal(storage.uploads.length, 1);
    assert.equal(storage.uploads[0]!.bucket, "subvision");
    assert.equal(storage.uploads[0]!.key, "outputs/u1");
    assert.equal(storage.uploads[0]!.filePath, path.join(tmpDir, "outputs", "u1.mp4"));

    // the frames were combined onto the downloaded video with the configured fps
    assert.equal(combineCalls.length, 1);
    assert.equal(combineCalls[0]!.fps, 30);
    assert.equal(combineCalls[0]!.videoPath, path.join(tmpDir, "videos", "u1"));
    assert.equal(combineCalls[0]!.framesDir, path.join(tmpDir, "frames", "u1"));
    assert.equal(combineCalls[0]!.outputPath, path.join(tmpDir, "outputs", "u1.mp4"));
  });

  it("downloads the video from the upload key derived from the job", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const storage = new FakeStorage();
    const { module } = makeModule(tmpDir, storage);

    await module.run({ uploadId: "u2", objectKey: "uploads/u2", segments });

    assert.equal(storage.downloads.length, 1);
    assert.equal(storage.downloads[0]!.key, "uploads/u2");
    assert.equal(storage.downloads[0]!.filePath, path.join(tmpDir, "videos", "u2"));
  });

  it("renders the job's segments unchanged", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const storage = new FakeStorage();
    const { module, renderFramesCalls } = makeModule(tmpDir, storage);

    await module.run({ uploadId: "u1", objectKey: "uploads/u1", segments });

    assert.deepEqual(renderFramesCalls[0]!.segments, segments);
  });

  it("derives the render option fps, not the caller", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const storage = new FakeStorage();
    const { module, combineCalls } = makeModule(tmpDir, storage, 24);

    await module.run({ uploadId: "u1", objectKey: "uploads/u1", segments });

    assert.equal(combineCalls[0]!.fps, 24);
  });

  it("rejects an object key that carries no id", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const { module } = makeModule(tmpDir, new FakeStorage());

    await assert.rejects(
      module.run({ uploadId: "u1", objectKey: "uploads/", segments }),
      /carries no id/
    );
  });

  it("rejects a job without segments", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const { module } = makeModule(tmpDir, new FakeStorage());

    await assert.rejects(
      module.run({ uploadId: "u1", objectKey: "uploads/u1", segments: [] }),
      /no segments/
    );
  });
});
