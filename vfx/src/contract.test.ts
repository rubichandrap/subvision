import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { VFX_JOBS_QUEUE, ContractError, parseVfxJob } from "./contract";

describe("vfx job contract", () => {
  it("consumes the queue the server publishes to", () => {
    // The server declares QueueName = "vfx_jobs" in server/internal/vfxjob.
    assert.equal(VFX_JOBS_QUEUE, "vfx_jobs");
  });

  it("parses a well-formed job, keeping every segment unchanged", () => {
    const raw = {
      uploadId: "u1",
      objectKey: "uploads/u1",
      segments: [
        { start: 0, end: 1.5, text: "hello" },
        { start: 1.5, end: 3, text: "world" },
      ],
      animationType: "karaoke",
    };

    const job = parseVfxJob(raw);

    assert.deepEqual(job, raw);
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

  it("rejects a payload that is not an object", () => {
    assert.throws(() => parseVfxJob("not a job"), ContractError);
    assert.throws(() => parseVfxJob(null), ContractError);
  });
});
