"""Decode long audio in overlapping chunks and stitch the words back together.

One Parakeet pass over a long file is erratic on music: moving where the audio
starts changes which sung words it hears, and fewer of them are heard again by
other decodes. Shorter passes are steadier (measurements in ADR-0012). This
module is the seam that hides that: a caller hands over the samples and a
``decode`` function and gets one ordered list of Timed Words on the whole
file's clock, however many passes it took.

It knows nothing about sherpa-onnx or numpy. ``decode`` is injected, takes a
slice of the samples, and returns words timed from the start of that slice;
the slice only needs ``len()`` and slicing, so a numpy view works unchanged.
Words are any frozen dataclass with ``text``, ``start`` and ``end`` (seconds).
"""

from __future__ import annotations

import math
from dataclasses import dataclass, replace
from typing import Callable, Sequence, TypeVar

W = TypeVar("W")

# A chunk is at least this long (or chunking is off).
MIN_CHUNK_SECONDS = 2.0

# What leaving a word unpaired costs when two chunks' words are lined up, in
# the same unit as the time between a pair's centers. Two copies of a word
# pair up only when that is cheaper than leaving both out, so when they lie
# closer than twice this (0.3 s). A frame is 0.08 s and chunks sit on their
# own frame grids, and a word right at a chunk's edge can be timed a couple of
# tenths late, so real copies of one word differ by a frame or two or three.
UNMATCHED_WORD_SECONDS = 0.15

# Where two chunks' words overlap, the earlier one gives way only down to this
# (half a frame); a word that would shrink further shares the overlap instead.
MIN_WORD_SECONDS = 0.04


@dataclass(frozen=True)
class ChunkOptions:
    """How long one decode pass may be, and how much neighbours share.

    ``chunk_seconds`` is the longest audio decoded in one pass; 0 turns
    chunking off, and otherwise it is at least ``MIN_CHUNK_SECONDS``.
    ``overlap_seconds`` is the audio two neighbouring chunks share, so every
    word is heard with context on both sides by at least one of them. It must
    stay under half a chunk.
    """

    chunk_seconds: float
    overlap_seconds: float

    def __post_init__(self) -> None:
        for name, value in (("chunk", self.chunk_seconds), ("overlap", self.overlap_seconds)):
            if not math.isfinite(value):
                raise ValueError(f"{name} seconds must be a finite number, got {value}")
        if self.chunk_seconds < 0:
            raise ValueError(f"chunk seconds must not be negative, got {self.chunk_seconds}")
        if self.chunk_seconds == 0:
            return
        if self.chunk_seconds < MIN_CHUNK_SECONDS:
            raise ValueError(
                f"chunk seconds must be 0 (off) or at least {MIN_CHUNK_SECONDS:g}, got {self.chunk_seconds}"
            )
        if not 0 <= self.overlap_seconds < self.chunk_seconds / 2:
            raise ValueError(
                f"overlap seconds must be at least 0 and under half the chunk "
                f"({self.chunk_seconds / 2:g}), got {self.overlap_seconds}"
            )

    @property
    def enabled(self) -> bool:
        return self.chunk_seconds > 0


def plan_chunks(n_samples: int, sample_rate: int, options: ChunkOptions) -> list[tuple[int, int]]:
    """Sample ranges to decode: one if the audio fits a pass, else the fewest
    equal chunks, each at most ``chunk_seconds``, that overlap by at least
    ``overlap_seconds`` and together cover the audio."""
    longest = round(options.chunk_seconds * sample_rate)
    if not options.enabled or n_samples <= longest:
        return [(0, n_samples)]
    overlap = round(options.overlap_seconds * sample_rate)
    count = math.ceil((n_samples - overlap) / (longest - overlap))
    length = math.ceil((n_samples + (count - 1) * overlap) / count)
    stride = length - overlap
    return [(i * stride, min(i * stride + length, n_samples)) for i in range(count)]


def decode_chunked(
    samples: Sequence,
    sample_rate: int,
    decode: Callable[[Sequence], list[W]],
    options: ChunkOptions,
) -> list[W]:
    """Timed Words for all of ``samples``, on the whole file's clock.

    Audio that fits one pass is decoded once and returned exactly as
    ``decode`` produced it.
    """
    spans = plan_chunks(len(samples), sample_rate, options)
    if len(spans) == 1:
        return decode(samples)
    chunks = []
    for first, last in spans:
        offset = first / sample_rate
        words = [replace(w, start=w.start + offset, end=w.end + offset) for w in decode(samples[first:last])]
        chunks.append((offset, last / sample_rate, words))
    return stitch(chunks)


def stitch(chunks: Sequence[tuple[float, float, Sequence[W]]]) -> list[W]:
    """Join per-chunk Timed Words into one list.

    ``chunks`` are ``(start, end, words)`` in time order, with word times
    already on the whole file's clock. Where neighbours overlap, the
    list crosses over from one chunk to the next at a word both heard,
    the one nearest the middle of the overlap, so each side of the crossing
    comes from the chunk with more audio around it and no word is heard
    twice. With no word in common it cuts at the middle instead, so a word
    only the far side of an overlap heard is dropped (ADR-0012).
    """
    merged: list[W] = []
    covered = 0.0
    for start, end, words in chunks:
        if start >= covered:
            merged.extend(words)
        else:
            merged = _cross_over(merged, list(words), start, covered)
        covered = max(covered, end)
    return _no_overlaps(merged)


def _cross_over(merged: list[W], incoming: list[W], overlap_start: float, overlap_end: float) -> list[W]:
    first = len(merged)
    while first > 0 and _center(merged[first - 1]) >= overlap_start:
        first -= 1
    tail = merged[first:]
    head = [w for w in incoming if _center(w) <= overlap_end]
    middle = (overlap_start + overlap_end) / 2

    pairs = _common_words(tail, head)
    if pairs:
        i, j = min(pairs, key=lambda pair: abs(_center(tail[pair[0]]) - middle))
        # The shared word itself comes from the chunk with more audio around it:
        # the earlier chunk hears what follows the word, the later one what precedes it.
        earlier_room = overlap_end - _center(tail[i])
        later_room = _center(head[j]) - overlap_start
        shared = tail[i] if earlier_room >= later_room else head[j]
        return merged[: first + i] + [shared] + incoming[j + 1 :]
    # Nothing shared: each side of the middle comes from the chunk with more
    # audio around it. Words only the far side of an overlap heard are dropped
    # even when the other chunk heard nothing there: they are the least
    # corroborated words of the list (ADR-0012).
    return merged[:first] + [w for w in tail if _center(w) < middle] + [
        w for w in incoming if _center(w) >= middle
    ]


def _common_words(tail: list[W], head: list[W]) -> list[tuple[int, int]]:
    """Index pairs of the words ``tail`` and ``head`` both heard, in order.

    It is the cheapest alignment: copies of a word pair up at the cost of the
    time between their centers, and a word left unpaired costs
    ``UNMATCHED_WORD_SECONDS``. Pricing the distance keeps a run of identical
    words ("no no no") lined up instance for instance; counting matches alone
    prefers an alignment shifted by one, which merges two instances.
    """
    n, m = len(tail), len(head)
    cost = [[0.0] * (m + 1) for _ in range(n + 1)]
    for i in range(n - 1, -1, -1):
        cost[i][m] = (n - i) * UNMATCHED_WORD_SECONDS
    for j in range(m - 1, -1, -1):
        cost[n][j] = (m - j) * UNMATCHED_WORD_SECONDS
    paired = [[False] * m for _ in range(n)]
    for i in range(n - 1, -1, -1):
        for j in range(m - 1, -1, -1):
            best = min(cost[i + 1][j], cost[i][j + 1]) + UNMATCHED_WORD_SECONDS
            if _same_text(tail[i], head[j]):
                together = cost[i + 1][j + 1] + abs(_center(tail[i]) - _center(head[j]))
                if together <= best:
                    best, paired[i][j] = together, True
            cost[i][j] = best
    pairs: list[tuple[int, int]] = []
    i = j = 0
    while i < n and j < m:
        if paired[i][j]:
            pairs.append((i, j))
            i += 1
            j += 1
        elif cost[i + 1][j] <= cost[i][j + 1]:
            i += 1
        else:
            j += 1
    return pairs


def _no_overlaps(words: list[W]) -> list[W]:
    """Words from two chunks can meet a frame out of step. Where one starts
    inside the one before it, the earlier word gives way: the list stays
    ordered with no overlap and the later word keeps its start, which is what
    a caption shows. If that would leave the earlier word with next to no
    duration, the two split the part they share instead (the Go client rejects
    a sentence whose only word has none)."""
    out: list[W] = []
    for word in words:
        if out and word.start < out[-1].end:
            previous = out[-1]
            boundary = word.start
            if boundary - previous.start < MIN_WORD_SECONDS:
                boundary = (word.start + min(previous.end, word.end)) / 2
            boundary = max(boundary, previous.start)
            out[-1] = replace(previous, end=boundary)
            word = replace(word, start=boundary, end=max(word.end, boundary))
        out.append(word)
    return out


def _same_text(a: W, b: W) -> bool:
    text = _normalize(a.text)
    return bool(text) and text == _normalize(b.text)


def _normalize(text: str) -> str:
    return "".join(ch for ch in text.casefold() if ch.isalnum())


def _center(word: W) -> float:
    return (word.start + word.end) / 2
