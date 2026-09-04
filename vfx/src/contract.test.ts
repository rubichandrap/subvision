import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { VFX_JOBS_QUEUE, ContractError, parseVfxJob } from "./contract";

const style = {
  fontFamily: "Montserrat",
  fontSizeScale: 1.2,
  color: "#FFFFFF",
  outlineWidth: 8,
  outlineColor: "#000000",
  bottomMargin: 0.1,
  background: "box",
  backgroundOpacity: 0.5,
  uppercase: true,
  highlightColor: "#FACC15",
};

describe("vfx job contract", () => {
  it("consumes the queue the server publishes to", () => {
    // The server declares QueueName = "vfx_jobs" in server/internal/vfxjob.
    assert.equal(VFX_JOBS_QUEUE, "vfx_jobs");
  });

  it("parses a well-formed job without an Edit Spec, keeping every segment unchanged", () => {
    const raw = {
      uploadId: "u1",
      objectKey: "uploads/u1",
      segments: [
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
      ],
    };

    const job = parseVfxJob(raw);

    assert.deepEqual(job, raw);
  });

  it("parses a job carrying an Edit Spec unchanged", () => {
    const raw = {
      uploadId: "u1",
      objectKey: "uploads/u1",
      segments: [
        {
          start: 0,
          end: 1.5,
          text: "hello",
          words: [{ text: "hello", start: 0, end: 1.5 }],
        },
      ],
      editSpec: {
        trim: { start: 2, end: 9 },
        frame: { preset: "9:16", ratio: 0.5625, zoom: 1.5, panX: -0.5, panY: 0 },
        animation: "pop",
        style,
        captions: { wordsPerPage: 4 },
      },
    };

    const job = parseVfxJob(raw);

    assert.deepEqual(job, raw);
  });

  it("accepts a free frame preset with a custom ratio", () => {
    const job = parseVfxJob({
      uploadId: "u1",
      objectKey: "uploads/u1",
      segments: [],
      editSpec: {
        trim: { start: 0, end: 0 },
        frame: { preset: "free", ratio: 1.37, zoom: 1, panX: 0, panY: 0 },
        animation: "fade",
        style,
      },
    });

    assert.equal(job.editSpec!.frame.preset, "free");
    assert.equal(job.editSpec!.trim.end, 0);
  });

  it("rejects a ratio contradicting the frame preset", () => {
    assert.throws(
      () =>
        parseVfxJob({
          uploadId: "u1",
          objectKey: "uploads/u1",
          segments: [],
          editSpec: {
            trim: { start: 0, end: 0 },
            frame: { preset: "9:16", ratio: 1.7777, zoom: 1, panX: 0, panY: 0 },
            animation: "fade",
            style,
          },
        }),
      /does not match preset/
    );
  });

  it("rejects an unknown animation", () => {
    assert.throws(
      () =>
        parseVfxJob({
          uploadId: "u1",
          objectKey: "uploads/u1",
          segments: [],
          editSpec: {
            trim: { start: 0, end: 0 },
            frame: { preset: "1:1", ratio: 1, zoom: 1, panX: 0, panY: 0 },
            animation: "random",
            style,
          },
        }),
      /animation/
    );
  });

  it("rejects an edit spec with a style field out of range", () => {
    assert.throws(
      () =>
        parseVfxJob({
          uploadId: "u1",
          objectKey: "uploads/u1",
          segments: [],
          editSpec: {
            trim: { start: 0, end: 0 },
            frame: { preset: "1:1", ratio: 1, zoom: 1, panX: 0, panY: 0 },
            animation: "fade",
            style: { ...style, outlineWidth: 64 },
          },
        }),
      /outlineWidth/
    );
  });

  it("rejects a trim window that ends before it starts", () => {
    assert.throws(
      () =>
        parseVfxJob({
          uploadId: "u1",
          objectKey: "uploads/u1",
          segments: [],
          editSpec: {
            trim: { start: 5, end: 4 },
            frame: { preset: "1:1", ratio: 1, zoom: 1, panX: 0, panY: 0 },
            animation: "fade",
            style,
          },
        }),
      /trim\.end/
    );
  });

  it("rejects a payload with a missing uploadId", () => {
    assert.throws(
      () => parseVfxJob({ objectKey: "uploads/u1", segments: [] }),
      ContractError
    );
  });

  it("rejects a payload with malformed segments", () => {
    assert.throws(
      () =>
        parseVfxJob({
          uploadId: "u1",
          objectKey: "uploads/u1",
          segments: [{ start: "zero", end: 1, text: "hello" }],
        }),
      /segments\[0\]\.start/
    );
  });

  it("rejects a segment without word timings", () => {
    assert.throws(
      () =>
        parseVfxJob({
          uploadId: "u1",
          objectKey: "uploads/u1",
          segments: [{ start: 0, end: 1.5, text: "hello" }],
        }),
      /segments\[0\]\.words/
    );
  });

  it("rejects a word timing with a non-numeric start", () => {
    assert.throws(
      () =>
        parseVfxJob({
          uploadId: "u1",
          objectKey: "uploads/u1",
          segments: [
            {
              start: 0,
              end: 1.5,
              text: "hello",
              words: [{ text: "hello", start: "0", end: 1.5 }],
            },
          ],
        }),
      /words\[0\]\.start/
    );
  });

  it("rejects a payload that is not an object", () => {
    assert.throws(() => parseVfxJob("not a job"), ContractError);
    assert.throws(() => parseVfxJob(null), ContractError);
  });

  it("defaults captions.wordsPerPage to 4 when the job carries no captions", () => {
    const job = parseVfxJob({
      uploadId: "u1",
      objectKey: "uploads/u1",
      segments: [],
      editSpec: {
        trim: { start: 0, end: 0 },
        frame: { preset: "1:1", ratio: 1, zoom: 1, panX: 0, panY: 0 },
        animation: "pop",
        style,
      },
    });

    assert.equal(job.editSpec!.captions.wordsPerPage, 4);
  });

  it("parses a job carrying captions.wordsPerPage unchanged", () => {
    const raw = {
      uploadId: "u1",
      objectKey: "uploads/u1",
      segments: [],
      editSpec: {
        trim: { start: 0, end: 0 },
        frame: { preset: "1:1", ratio: 1, zoom: 1, panX: 0, panY: 0 },
        animation: "pop",
        style,
        captions: { wordsPerPage: 6 },
      },
    };

    const job = parseVfxJob(raw);

    assert.deepEqual(job, raw);
  });

  it("rejects captions.wordsPerPage outside 2-8", () => {
    for (const wordsPerPage of [1, 9]) {
      assert.throws(
        () =>
          parseVfxJob({
            uploadId: "u1",
            objectKey: "uploads/u1",
            segments: [],
            editSpec: {
              trim: { start: 0, end: 0 },
              frame: { preset: "1:1", ratio: 1, zoom: 1, panX: 0, panY: 0 },
              animation: "pop",
              style,
              captions: { wordsPerPage },
            },
          }),
        /wordsPerPage/
      );
    }
  });

  it("rejects a non-integer captions.wordsPerPage", () => {
    assert.throws(
      () =>
        parseVfxJob({
          uploadId: "u1",
          objectKey: "uploads/u1",
          segments: [],
          editSpec: {
            trim: { start: 0, end: 0 },
            frame: { preset: "1:1", ratio: 1, zoom: 1, panX: 0, panY: 0 },
            animation: "pop",
            style,
            captions: { wordsPerPage: 4.5 },
          },
        }),
      /wordsPerPage/
    );
  });
});
