import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, it } from "node:test";

import { ISegment, IWord } from "../types";
import { activePageWords, activeSegment, onsetStart } from "./shared";

// The acceptance fixture (issue #23, ADR-0006; gated on detected speech per
// ADR-0007, #27 reverted to stock third-party code): real Transcription
// Segments for a 24-second clip — leading silence, then speech (whisper.cpp's
// jfk sample), then trailing silence — transcribed by the production
// transcriber and stored in server/testdata/onset-fixture. Regenerate with
// `SPEECH_GATING=true go run ./cmd/onsetfixture <model> <wav> <out>` when the
// model or the vendored build changes.
//
// Measured finding baked into this fixture: the physical speech onset is
// 3.33 s (ffmpeg silencedetect −30 dB on the fixture wav — the −35 dB
// crossing at 3.10 s is the sample's room-tone ramp, not speech), and the
// first reported word sits at 3.34 s. The transcriber runs ffmpeg
// silencedetect itself and decodes only the speech windows, so the decoder
// never sees the leading silence (no first-token clamping into it) and the
// reported timings stay on the real timeline.
const segments: ISegment[] = JSON.parse(
  readFileSync(
    resolve(process.cwd(), "../server/testdata/onset-fixture/segments.json"),
    "utf8"
  )
) as ISegment[];

function texts(words: IWord[]): string[] {
  return words.map((w) => w.text);
}

describe("onset fixture (real whisper output)", () => {
  it("carries word-bearing segments with words inside their window", () => {
    assert.ok(segments.length > 0);
    for (const segment of segments) {
      assert.ok(segment.end > segment.start);
      for (const word of segment.words) {
        assert.ok(word.start >= segment.start - 1e-9);
        assert.ok(word.end <= segment.end + 1e-9);
      }
    }
  });

  it("shows no words before each segment's onset", () => {
    for (const segment of segments) {
      if (segment.words.length === 0) continue;
      assert.deepEqual(
        texts(activePageWords(segment, onsetStart(segment) - 0.001)),
        []
      );
    }
  });

  it("shows the first page exactly at each segment's onset", () => {
    for (const segment of segments) {
      if (segment.words.length === 0) continue;
      const want = segment.words.slice(0, 4).map((w) => w.text);
      assert.deepEqual(texts(activePageWords(segment, onsetStart(segment))), want);
    }
  });

  it("gates the whole timeline: nothing active before the first onset", () => {
    const firstOnset = Math.min(...segments.map(onsetStart));
    assert.equal(activeSegment(segments, firstOnset - 0.001), undefined);
    assert.ok(activeSegment(segments, firstOnset));
  });
});
