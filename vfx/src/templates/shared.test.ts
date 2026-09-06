import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { DEFAULT_WORDS_PER_PAGE } from "../contract";
import { ISegment, IWord } from "../types";
import { activePageWords, activeSegment, onsetStart } from "./shared";

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

// A segment whose window opens `lead` seconds before its first word starts —
// the leading-silence shape real whisper output has.
function makeLeadSegment(count: number, lead: number): ISegment {
  const segment = makeSegment(count);
  segment.start = 0;
  segment.words[0]!.start = lead;
  return segment;
}

function texts(words: IWord[]): string[] {
  return words.map((w) => w.text);
}

describe("activeSegment", () => {
  it("stays inactive before the first word starts (onset gate)", () => {
    const segment = makeLeadSegment(6, 1);
    assert.equal(activeSegment([segment], 0.5), undefined);
    assert.equal(activeSegment([segment], 0.999), undefined);
  });

  it("becomes active exactly at the first word start (onset edge)", () => {
    const segment = makeLeadSegment(6, 1);
    assert.equal(activeSegment([segment], 1), segment);
  });

  it("keeps the segment window: inactive before start and at end", () => {
    const segment = makeSegment(3);
    assert.equal(activeSegment([segment], -0.1), undefined);
    assert.equal(activeSegment([segment], segment.end), undefined);
  });

  it("treats wordless segments differently from silent word-driven ones", () => {
    // Same window, same time: the wordless segment (a music or sound marker)
    // is active from its own start; the word-driven one stays dark until
    // its onset. Any gate on wordless segments would break this.
    const wordless = makeSegment(0);
    const wordDriven = makeLeadSegment(6, 1);
    assert.equal(activeSegment([wordless], 0.5), wordless);
    assert.equal(activeSegment([wordDriven], 0.5), undefined);
  });
});

describe("onsetStart", () => {
  it("is the first word's start for a word-driven segment", () => {
    const segment = makeSegment(6);
    segment.words[0]!.start = 1;
    assert.equal(onsetStart(segment), 1);
  });

  it("is the segment start for a wordless segment", () => {
    const wordless = makeSegment(0);
    wordless.start = 2.5;
    assert.equal(onsetStart(wordless), 2.5);
  });
});

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

  it("leaves wordless segments unaffected at any time (no onset gate)", () => {
    const wordless = makeSegment(0);
    assert.deepEqual(activePageWords(wordless, 0), []);
    assert.deepEqual(activePageWords(wordless, 5), []);
  });

  it("yields no words before the first word starts (onset gate)", () => {
    const segment = makeLeadSegment(6, 1);
    assert.deepEqual(texts(activePageWords(segment, 0.5, 4)), []);
    assert.deepEqual(texts(activePageWords(segment, 0.999, 4)), []);
  });

  it("shows the first page exactly at first word start (onset edge)", () => {
    const segment = makeLeadSegment(6, 1);
    assert.deepEqual(texts(activePageWords(segment, 1, 4)), ["w0", "w1", "w2", "w3"]);
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
