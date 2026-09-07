import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, it } from "node:test";

import { ISegment, IWord } from "../types";
import { activePageWords, activeSegment, onsetStart } from "./shared";

// The acceptance fixture (issue #23, ADR-0006): real Transcription Segments
// for a 24-second clip — leading silence, then speech (whisper.cpp's jfk
// sample), then trailing silence — transcribed by the production transcriber
// in one pass over the whole audio and stored in
// server/testdata/onset-fixture. Regenerate with
// `go run ./cmd/onsetfixture <model> <wav> <out>` when the model or the
// vendored build changes.
//
// Ungated decode hallucinates "And so my" over the leading silence (0–1.75 s)
// — the known one-pass behavior the onset gate (ADR-0006) covers on the render
// side: captions stay hidden until each segment's first word starts.
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
