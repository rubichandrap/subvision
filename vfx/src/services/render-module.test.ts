import assert from "node:assert/strict";
import fs from "fs";
import os from "os";
import path from "path";
import { describe, it } from "node:test";

import {
  CombinePlan,
  RenderModule,
  RenderModuleConfig,
  VideoProber,
  maxSegmentEnd,
  planFrame,
  shiftSegments,
} from "./render-module";
import { ISegment } from "../types";

const segments: ISegment[] = [
  {
    start: 0,
    end: 1.5,
    text: "hello",
    words: [{ text: "hello", start: 0, end: 1.5 }],
  },
  {
    start: 1.5,
    end: 3,
    text: "world",
    words: [{ text: "world", start: 1.5, end: 3 }],
  },
];

const style = {
  fontFamily: "Montserrat" as const,
  fontSizeScale: 1,
  color: "#FFFFFF",
  outlineWidth: 0,
  outlineColor: "#000000",
  bottomMargin: 0.12,
  background: "none" as const,
  backgroundOpacity: 0.5,
  uppercase: false,
  highlightColor: "#FACC15",
};

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

function makeModule(
  tmpDir: string,
  storage: FakeStorage,
  fps = 30,
  probe: VideoProber = async () => ({ width: 1920, height: 1080, duration: 60 })
) {
  const config: RenderModuleConfig = {
    tmpDir,
    bucket: "subvision",
    options: { fps, width: 1920, height: 1080 },
    template: "karaoke",
  };
  const renderCalls: Array<{
    segments: ISegment[];
    framesDir: string;
    width: number;
    height: number;
    fps: number;
    template: string;
  }> = [];
  const combineCalls: Array<{
    videoPath: string;
    framesDir: string;
    outputPath: string;
    fps: number;
    plan: CombinePlan;
  }> = [];
  const renderFrames = async (request: {
    segments: ISegment[];
    framesDir: string;
    width: number;
    height: number;
    fps: number;
    template: string;
  }) => {
    renderCalls.push(request);
  };
  const combineFrames = async (
    videoPath: string,
    framesDir: string,
    outputPath: string,
    frameRate: number,
    plan: CombinePlan
  ) => {
    combineCalls.push({ videoPath, framesDir, outputPath, fps: frameRate, plan });
  };
  const module = new RenderModule(storage, renderFrames, combineFrames, config, probe);
  return { module, renderCalls, combineCalls };
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

  it("renders the job's segments unchanged at the configured dimensions with the fallback template", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const storage = new FakeStorage();
    const { module, renderCalls } = makeModule(tmpDir, storage);

    await module.run({ uploadId: "u1", objectKey: "uploads/u1", segments });

    assert.deepEqual(renderCalls[0]!.segments, segments);
    assert.equal(renderCalls[0]!.width, 1920);
    assert.equal(renderCalls[0]!.height, 1080);
    assert.equal(renderCalls[0]!.template, "karaoke");
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

describe("render module with an Edit Spec", () => {
  const editSpec = {
    trim: { start: 10, end: 20 },
    frame: { preset: "9:16", ratio: 9 / 16, zoom: 1.5, panX: -1, panY: 0 },
    animation: "pop" as const,
    style,
  };

  it("probes the video and renders the overlay at the frame's dimensions with the job's animation", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const storage = new FakeStorage();
    const probedPaths: string[] = [];
    const { module, renderCalls } = makeModule(tmpDir, storage, 30, async (videoPath) => {
      probedPaths.push(videoPath);
      return { width: 1920, height: 1080, duration: 60 };
    });

    await module.run({ uploadId: "u1", objectKey: "uploads/u1", segments, editSpec });

    assert.equal(probedPaths.length, 1);
    assert.equal(probedPaths[0], path.join(tmpDir, "videos", "u1"));
    assert.equal(renderCalls[0]!.template, "pop");
    assert.equal(renderCalls[0]!.width, 606);
    assert.equal(renderCalls[0]!.height, 1076);
    assert.equal(renderCalls[0]!.fps, 30);
  });

  it("shifts the segments into the trim window and cuts the video accordingly", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const storage = new FakeStorage();
    const { module, renderCalls, combineCalls } = makeModule(tmpDir, storage);

    // A segment outside the trim window is dropped; one straddling it is
    // clipped, and its words are clipped and dropped with it.
    await module.run({
      uploadId: "u1",
      objectKey: "uploads/u1",
      segments: [
        { start: 2, end: 5, text: "before", words: [{ text: "before", start: 2, end: 5 }] },
        {
          start: 12,
          end: 16,
          text: "inside",
          words: [
            { text: "in", start: 12, end: 14 },
            { text: "side", start: 14, end: 16 },
          ],
        },
        {
          start: 18,
          end: 25,
          text: "straddles",
          words: [
            { text: "strad", start: 18, end: 19.5 },
            { text: "dles", start: 19.5, end: 25 },
          ],
        },
      ],
      editSpec,
    });

    assert.deepEqual(renderCalls[0]!.segments, [
      {
        start: 2,
        end: 6,
        text: "inside",
        words: [
          { text: "in", start: 2, end: 4 },
          { text: "side", start: 4, end: 6 },
        ],
      },
      {
        start: 8,
        end: 10,
        text: "straddles",
        words: [
          { text: "strad", start: 8, end: 9.5 },
          { text: "dles", start: 9.5, end: 10 },
        ],
      },
    ]);
    assert.equal(combineCalls[0]!.plan.seekStart, 10);
    assert.equal(combineCalls[0]!.plan.seekDuration, 10);
    // The overlay is disabled after the last shifted segment ends.
    assert.equal(combineCalls[0]!.plan.overlayUntil, 10);
  });

  it("applies the crop-to-fill filters derived from the probed dimensions", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const storage = new FakeStorage();
    const { module, combineCalls } = makeModule(tmpDir, storage);

    await module.run({ uploadId: "u1", objectKey: "uploads/u1", segments, editSpec });

    const filters = combineCalls[0]!.plan.videoFilters;
    assert.equal(filters.length, 2);
    assert.equal(filters[0], "scale=2869:1614");
    assert.equal(filters[1], "crop=606:1076:0:269"); // panX -1 pins the window left
  });

  it("renders without a trim end to the end of the video", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const storage = new FakeStorage();
    const { module, combineCalls } = makeModule(tmpDir, storage);

    await module.run({
      uploadId: "u1",
      objectKey: "uploads/u1",
      segments,
      editSpec: {
        ...editSpec,
        trim: { start: 5, end: 0 },
      },
    });

    assert.equal(combineCalls[0]!.plan.seekStart, 5);
    assert.equal(combineCalls[0]!.plan.seekDuration, null);
  });

  it("surfaces a probe failure loudly", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "render-module-"));
    const storage = new FakeStorage();
    const { module } = makeModule(tmpDir, storage, 30, async () => {
      throw new Error("ffprobe exploded");
    });

    await assert.rejects(
      module.run({ uploadId: "u1", objectKey: "uploads/u1", segments, editSpec }),
      /ffprobe exploded/
    );
  });
});

describe("planFrame", () => {
  it("covers a landscape source into a portrait frame without upscaling past the source", () => {
    const plan = planFrame(
      { preset: "9:16", ratio: 9 / 16, zoom: 1, panX: 0, panY: 0 },
      3840,
      2160
    );
    assert.equal(plan.width, 1080);
    assert.equal(plan.height, 1920);
    assert.equal(plan.filters[0], "scale=3413:1920");
  });

  it("shrinks the frame to the source rather than upscaling it", () => {
    const plan = planFrame(
      { preset: "1:1", ratio: 1, zoom: 1, panX: 0, panY: 0 },
      1920,
      1080
    );
    assert.equal(plan.width, 1080);
    assert.equal(plan.height, 1080);
    assert.equal(plan.filters[0], "scale=1920:1080");
  });

  it("centers the crop window at neutral pan", () => {
    const plan = planFrame(
      { preset: "1:1", ratio: 1, zoom: 1, panX: 0, panY: 0 },
      1920,
      1080
    );
    assert.equal(plan.filters[1], "crop=1080:1080:420:0");
  });

  it("pins the crop window to the edges at extreme pan", () => {
    const left = planFrame(
      { preset: "1:1", ratio: 1, zoom: 1, panX: -1, panY: 0 },
      1920,
      1080
    );
    assert.equal(left.filters[1], "crop=1080:1080:0:0");

    const right = planFrame(
      { preset: "1:1", ratio: 1, zoom: 1, panX: 1, panY: 0 },
      1920,
      1080
    );
    assert.equal(right.filters[1], "crop=1080:1080:840:0");
  });

  it("zooms past cover before panning", () => {
    const plan = planFrame(
      { preset: "1:1", ratio: 1, zoom: 2, panX: 0, panY: 0 },
      1920,
      1080
    );
    assert.equal(plan.filters[0], "scale=3840:2160");
    assert.equal(plan.filters[1], "crop=1080:1080:1380:540");
  });

  it("rejects probes without usable dimensions", () => {
    assert.throws(
      () => planFrame({ preset: "1:1", ratio: 1, zoom: 1, panX: 0, panY: 0 }, 0, 1080),
      /invalid dimensions/
    );
  });
});

describe("shiftSegments", () => {
  it("returns empty when no segment intersects the window", () => {
    assert.deepEqual(shiftSegments(segments, { start: 10, end: 20 }), []);
  });

  it("clips segments that straddle the window edges", () => {
    const shifted = shiftSegments(
      [
        {
          start: 1,
          end: 6,
          text: "straddles start",
          words: [{ text: "straddles start", start: 1, end: 6 }],
        },
      ],
      { start: 3, end: 0 }
    );
    assert.deepEqual(shifted, [
      {
        start: 0,
        end: 3,
        text: "straddles start",
        words: [{ text: "straddles start", start: 0, end: 3 }],
      },
    ]);
  });

  it("renders to the segment end when the trim end is open", () => {
    const shifted = shiftSegments(segments, { start: 1, end: 0 });
    // "hello" straddles the window start and is clipped to it; "world" shifts in.
    assert.deepEqual(shifted, [
      {
        start: 0,
        end: 0.5,
        text: "hello",
        words: [{ text: "hello", start: 0, end: 0.5 }],
      },
      {
        start: 0.5,
        end: 2,
        text: "world",
        words: [{ text: "world", start: 0.5, end: 2 }],
      },
    ]);
  });

  it("drops words outside the window and clips words at its edges", () => {
    const shifted = shiftSegments(
      [
        {
          start: 2,
          end: 6,
          text: "gone kept clipped",
          words: [
            { text: "gone", start: 2, end: 3 },
            { text: "kept", start: 3, end: 5 },
            { text: "clipped", start: 5, end: 6 },
          ],
        },
      ],
      { start: 3, end: 5.5 }
    );
    // "gone" ends exactly at the window start and contributes nothing,
    // "kept" shifts in whole, "clipped" clamps to the window end.
    assert.deepEqual(shifted, [
      {
        start: 0,
        end: 2.5,
        text: "gone kept clipped",
        words: [
          { text: "kept", start: 0, end: 2 },
          { text: "clipped", start: 2, end: 2.5 },
        ],
      },
    ]);
  });
});

describe("maxSegmentEnd", () => {
  it("is null without segments", () => {
    assert.equal(maxSegmentEnd([]), null);
  });

  it("is the latest segment end", () => {
    assert.equal(maxSegmentEnd(segments), 3);
  });
});
