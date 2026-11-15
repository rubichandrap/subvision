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

# Two chunks heard the same word when the text matches (ignoring case and
# punctuation) and their centers lie this close together. A frame is 0.08 s
# and chunks sit on their own frame grids, so real copies of one word differ
# by a frame or two; the same word sung again is seconds away.
SAME_WORD_WITHIN_SECONDS = 0.4


@dataclass(frozen=True)
class ChunkOptions:
    """How long one decode pass may be, and how much neighbours share.

    ``chunk_seconds`` is the longest audio decoded in one pass; 0 turns
    chunking off. ``overlap_seconds`` is the audio two neighbouring chunks
    share, so every word is heard with context on both sides by at least one
    of them. It must stay under half a chunk.
    """

    chunk_seconds: float
    overlap_seconds: float

    def __post_init__(self) -> None:
        if self.chunk_seconds < 0:
            raise ValueError(f"chunk seconds must not be negative, got {self.chunk_seconds}")
        if self.chunk_seconds > 0 and not 0 <= self.overlap_seconds < self.chunk_seconds / 2:
            raise ValueError(
                f"overlap seconds must be at least 0 and under half the chunk "
                f"({self.chunk_seconds / 2}), got {self.overlap_seconds}"
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
    """Join per-chunk words into one transcript.

    ``chunks`` are ``(start, end, words)`` in time order, with word times
    already on the whole file's clock. Where neighbours overlap, the
    transcript crosses over from one chunk to the next at a word both heard,
    the one nearest the middle of the overlap, so each side of the crossing
    comes from the chunk with more audio around it and no word is repeated
    or dropped. With no word in common it cuts at the middle instead.
    """
    merged: list[W] = []
    covered = 0.0
    for start, end, words in chunks:
        if not merged or start >= covered:
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
        # The shared word itself comes from whichever chunk has more room around it.
        room_before = overlap_end - _center(tail[i])
        room_after = _center(head[j]) - overlap_start
        shared = tail[i] if room_before >= room_after else head[j]
        return merged[: first + i] + [shared] + incoming[j + 1 :]
    # Nothing shared: each side of the middle comes from the chunk with more
    # audio around it. Words only the far side of an overlap heard are dropped
    # even when the other chunk heard nothing there: they are the least
    # corroborated words in the transcript (ADR-0012).
    return merged[:first] + [w for w in tail if _center(w) < middle] + [
        w for w in incoming if _center(w) >= middle
    ]


def _common_words(tail: list[W], head: list[W]) -> list[tuple[int, int]]:
    """Longest in-order run of words ``tail`` and ``head`` both heard."""
    n, m = len(tail), len(head)
    same = [[_same_word(tail[i], head[j]) for j in range(m)] for i in range(n)]
    best = [[0] * (m + 1) for _ in range(n + 1)]
    for i in range(n - 1, -1, -1):
        for j in range(m - 1, -1, -1):
            best[i][j] = best[i + 1][j + 1] + 1 if same[i][j] else max(best[i + 1][j], best[i][j + 1])
    pairs: list[tuple[int, int]] = []
    i = j = 0
    while i < n and j < m:
        if same[i][j] and best[i][j] == best[i + 1][j + 1] + 1:
            pairs.append((i, j))
            i += 1
            j += 1
        elif best[i + 1][j] >= best[i][j + 1]:
            i += 1
        else:
            j += 1
    return pairs


def _no_overlaps(words: list[W]) -> list[W]:
    """Words from two chunks can meet a frame out of step. The earlier word
    gives way, so the list stays ordered and no word starts inside another."""
    out: list[W] = []
    for word in words:
        if out and word.start < out[-1].end:
            previous = out[-1]
            cut = max(previous.start, word.start)
            out[-1] = replace(previous, end=cut)
            if word.start < cut:
                word = replace(word, start=cut, end=max(word.end, cut))
        out.append(word)
    return out


def _same_word(a: W, b: W) -> bool:
    text = _normalize(a.text)
    return bool(text) and text == _normalize(b.text) and abs(_center(a) - _center(b)) <= SAME_WORD_WITHIN_SECONDS


def _normalize(text: str) -> str:
    return "".join(ch for ch in text.casefold() if ch.isalnum())


def _center(word: W) -> float:
    return (word.start + word.end) / 2
