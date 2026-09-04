import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { ISegment, IWord } from "../types";
import { DEFAULT_WORDS_PER_PAGE, activePageWords } from "./shared";

// Synthetic segment: word i spans [i, i + 0.8], segment holds `trailing`
// seconds past the last word end (trailing pauses are real whisper output).
function makeSegment(count: number, trailing = 1): ISegment {
  const words: IWord[] = Array.from({ length: count }, (_, i) => ({
    text: `w${i}`,
    start: i,
    end: i + 0.8,
  }));
  const end = count > 0 ? count - 1 + 0.8 + trailing : trailing;
  return { start: 0, end, text: words.map((w) => w.text).join(" "), words };
}

function texts(words: IWord[]): string[] {
  return words.map((w) => w.text);
}

describe("activePageWords", () => {
  it("defaults to 4 words per page", () => {
    assert.equal(DEFAULT_WORDS_PER_PAGE, 4);
    const segment = makeSegment(6);
    assert.deepEqual(texts(activePageWords(segment, 0)), ["w0", "w1", "w2", "w3"]);
  });

  it("switches to the next page exactly when its first word starts", () => {
    const segment = makeSegment(8);
    assert.deepEqual(texts(activePageWords(segment, 3.99, 4)), ["w0", "w1", "w2", "w3"]);
    assert.deepEqual(texts(activePageWords(segment, 4, 4)), ["w4", "w5", "w6", "w7"]);
  });

  it("holds the current page through gaps between pages", () => {
    // w3 ends at 3.8, w4 starts at 4: between them page 0 stays up.
    const segment = makeSegment(8);
    assert.deepEqual(texts(activePageWords(segment, 3.9, 4)), ["w0", "w1", "w2", "w3"]);
  });

  it("holds the last page until the segment ends", () => {
    const segment = makeSegment(6);
    const lastEnd = 5 + 0.8;
    assert(segment.end > lastEnd);
    assert.deepEqual(texts(activePageWords(segment, lastEnd + 0.5, 4)), ["w4", "w5"]);
  });

  it("returns every word when the page size covers the segment", () => {
    const segment = makeSegment(3);
    assert.deepEqual(texts(activePageWords(segment, 2, 10)), ["w0", "w1", "w2"]);
  });

  it("returns no words for a segment without words", () => {
    assert.deepEqual(activePageWords(makeSegment(0), 0), []);
  });

  it("shows the first page before the first word starts", () => {
    const segment = makeSegment(6);
    segment.words[0]!.start = 1;
    assert.deepEqual(texts(activePageWords(segment, 0.5, 4)), ["w0", "w1", "w2", "w3"]);
  });

  it("partitions the segment: every word on exactly one page", () => {
    const segment = makeSegment(10);
    const seen: string[] = [];
    for (const page of [0, 1, 2]) {
      seen.push(...texts(activePageWords(segment, page * 4, 4)));
    }
    assert.deepEqual(seen, texts(segment.words));
  });
});
