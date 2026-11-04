#!/usr/bin/env python3
"""Compare two Subvision-shaped transcript JSON payloads.

This is a disposable spike helper, not production code. It computes transcript
agreement between two payloads and word-timing deltas for words that match
after punctuation/case normalization. A separate hand transcript can be
supplied with --reference to compute true WER against that text.
"""

from __future__ import annotations

import argparse
import difflib
import json
import math
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, Sequence


@dataclass(frozen=True)
class Word:
    text: str
    start: float
    end: float


@dataclass(frozen=True)
class TimingDelta:
    text: str
    baseline_start: float
    candidate_start: float
    start_delta: float
    baseline_end: float
    candidate_end: float
    end_delta: float


def flatten_words(payload: dict[str, object] | list[object]) -> list[Word]:
    if isinstance(payload, list):
        segments: list[object] = payload
    else:
        raw_segments = payload.get("segments", [])
        if not isinstance(raw_segments, list):
            raise ValueError("payload['segments'] must be an array")
        segments = raw_segments

    words: list[Word] = []
    for segment in segments:
        if not isinstance(segment, dict):
            raise ValueError("each segment must be an object")
        raw_words = segment.get("words", [])
        if not isinstance(raw_words, list):
            raise ValueError("segment['words'] must be an array")
        for raw in raw_words:
            if not isinstance(raw, dict):
                raise ValueError("each word must be an object")
            words.append(
                Word(
                    text=str(raw["text"]),
                    start=float(raw["start"]),
                    end=float(raw["end"]),
                )
            )
    return words


def payload_text(payload: dict[str, object] | list[object]) -> str:
    if isinstance(payload, dict):
        text = payload.get("text")
        if isinstance(text, str):
            return text.strip()
    if isinstance(payload, list):
        texts = [str(segment.get("text", "")).strip() for segment in payload if isinstance(segment, dict)]
        return " ".join(text for text in texts if text)
    return ""


def normalize_word(text: str) -> str:
    return re.sub(r"[^\w]+", "", text, flags=re.UNICODE).lower()


def align_words(
    baseline: Sequence[Word], candidate: Sequence[Word]
) -> list[tuple[Word, Word]]:
    matcher = difflib.SequenceMatcher(
        a=[normalize_word(word.text) for word in baseline],
        b=[normalize_word(word.text) for word in candidate],
        autojunk=False,
    )
    pairs: list[tuple[Word, Word]] = []
    for block in matcher.get_matching_blocks():
        for offset in range(block.size):
            pairs.append((baseline[block.a + offset], candidate[block.b + offset]))
    return pairs


def timing_deltas(pairs: Iterable[tuple[Word, Word]]) -> list[TimingDelta]:
    deltas: list[TimingDelta] = []
    for baseline_word, candidate_word in pairs:
        deltas.append(
            TimingDelta(
                text=baseline_word.text,
                baseline_start=baseline_word.start,
                candidate_start=candidate_word.start,
                start_delta=candidate_word.start - baseline_word.start,
                baseline_end=baseline_word.end,
                candidate_end=candidate_word.end,
                end_delta=candidate_word.end - baseline_word.end,
            )
        )
    return deltas


def percentile(values: Sequence[float], percentile: float) -> float | None:
    if not values:
        return None
    ordered = sorted(values)
    index = min(len(ordered) - 1, max(0, int(round(percentile * (len(ordered) - 1)))))
    return ordered[index]


def timing_summary(deltas: Sequence[TimingDelta]) -> dict[str, object]:
    def stats(field: str) -> dict[str, float | None]:
        values = [abs(getattr(delta, field)) for delta in deltas]
        raw = [getattr(delta, field) for delta in deltas]
        return {
            "mae": sum(values) / len(values) if values else None,
            "p95": percentile(values, 0.95),
            "worst": max(values) if values else None,
            "bias": sum(raw) / len(raw) if raw else None,
        }

    return {"start": stats("start_delta"), "end": stats("end_delta")}


def levenshtein(reference: Sequence[str], hypothesis: Sequence[str]) -> int:
    previous = list(range(len(hypothesis) + 1))
    for row, reference_word in enumerate(reference, 1):
        current = [row]
        for column, hypothesis_word in enumerate(hypothesis, 1):
            cost = 0 if reference_word == hypothesis_word else 1
            current.append(
                min(
                    previous[column] + 1,
                    current[column - 1] + 1,
                    previous[column - 1] + cost,
                )
            )
        previous = current
    return previous[-1]


def text_words(text: str) -> list[str]:
    return [normalize_word(word) for word in text.split() if normalize_word(word)]


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8").strip()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("baseline", type=Path, help="baseline transcript JSON")
    parser.add_argument("candidate", type=Path, help="candidate transcript JSON")
    parser.add_argument(
        "--reference",
        type=Path,
        help="optional plain-text hand transcript for true WER",
    )
    parser.add_argument("--json", action="store_true", help="emit JSON instead of text")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    baseline_payload = json.loads(args.baseline.read_text(encoding="utf-8"))
    candidate_payload = json.loads(args.candidate.read_text(encoding="utf-8"))
    baseline_words = flatten_words(baseline_payload)
    candidate_words = flatten_words(candidate_payload)
    pairs = align_words(baseline_words, candidate_words)
    deltas = timing_deltas(pairs)

    baseline_text = payload_text(baseline_payload)
    candidate_text = payload_text(candidate_payload)
    agreement_edits = levenshtein(text_words(baseline_text), text_words(candidate_text))
    agreement_denominator = max(1, len(text_words(baseline_text)))
    agreement = agreement_edits / agreement_denominator

    reference_wer = None
    if args.reference:
        reference = text_words(read_text(args.reference))
        candidate = text_words(candidate_text)
        reference_wer = levenshtein(reference, candidate) / max(1, len(reference))

    report = {
        "baseline_words": len(baseline_words),
        "candidate_words": len(candidate_words),
        "matched_words": len(deltas),
        "transcript_disagreement": agreement,
        "reference_wer": reference_wer,
        "timing": timing_summary(deltas),
        "deltas": [delta.__dict__ for delta in deltas],
    }

    if args.json:
        print(json.dumps(report, ensure_ascii=False, indent=2))
        return 0

    print(f"baseline words: {len(baseline_words)}")
    print(f"candidate words: {len(candidate_words)}")
    print(f"matched words: {len(deltas)}")
    print(f"transcript disagreement: {agreement:.3f}")
    if reference_wer is not None:
        print(f"reference WER: {reference_wer:.3f}")
    timing = report["timing"]
    for field in ("start", "end"):
        stats = timing[field]
        mae = stats["mae"]
        p95 = stats["p95"]
        worst = stats["worst"]
        bias = stats["bias"]
        mae_text = f"{mae:.3f}s" if isinstance(mae, float) else "n/a"
        p95_text = f"{p95:.3f}s" if isinstance(p95, float) else "n/a"
        worst_text = f"{worst:.3f}s" if isinstance(worst, float) else "n/a"
        bias_text = f"{bias:+.3f}s" if isinstance(bias, float) else "n/a"
        print(
            f"{field} delta: MAE {mae_text}, p95 {p95_text}, "
            f"worst {worst_text}, bias {bias_text}"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
