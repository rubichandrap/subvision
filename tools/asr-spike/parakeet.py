#!/usr/bin/env python3
"""Run NVIDIA Parakeet TDT v3 through sherpa-onnx on a 16 kHz mono WAV.

This is a disposable spike tool, not a production dependency. It writes a JSON
payload containing recognized text, Subvision-compatible Segments, and the raw
sherpa-onnx token timestamps/durations used to derive those words.
"""

from __future__ import annotations

import argparse
import contextlib
import json
import os
import re
import sys
import wave
from dataclasses import asdict, dataclass
from typing import Iterable

import numpy as np
import sherpa_onnx


@dataclass(frozen=True)
class Word:
    text: str
    start: float
    end: float


@dataclass(frozen=True)
class Token:
    text: str
    start: float
    end: float


def load_wav(path: str) -> tuple[np.ndarray, int]:
    """Read 16 kHz mono PCM s16le WAV as float32 samples."""
    with contextlib.closing(wave.open(path, "rb")) as reader:
        if reader.getframerate() != 16000:
            raise ValueError(f"expected 16000 Hz WAV, got {reader.getframerate()} Hz")
        if reader.getnchannels() != 1:
            raise ValueError(f"expected mono WAV, got {reader.getnchannels()} channels")
        if reader.getsampwidth() != 2:
            raise ValueError(
                f"expected 16-bit PCM WAV, got {reader.getsampwidth() * 8}-bit"
            )
        sample_rate = reader.getframerate()
        raw = reader.readframes(reader.getnframes())

    samples = np.frombuffer(raw, dtype=np.int16).astype(np.float32) / 32768.0
    return samples, sample_rate


def tokens_from_result(result) -> list[Token]:
    """Convert sherpa-onnx token timestamps/durations into ordered Tokens."""
    tokens = getattr(result, "tokens", None)
    timestamps = getattr(result, "timestamps", None)
    durations = getattr(result, "durations", None)
    if tokens is None or timestamps is None or durations is None:
        raise ValueError(
            "sherpa-onnx result did not expose tokens, timestamps, and durations; "
            "is the model a nemo_transducer with timing support?"
        )
    if not (len(tokens) == len(timestamps) == len(durations)):
        raise ValueError(
            "sherpa-onnx returned mismatched token/timing lengths "
            f"({len(tokens)} tokens, {len(timestamps)} timestamps, "
            f"{len(durations)} durations)"
        )

    out: list[Token] = []
    for text, start, duration in zip(tokens, timestamps, durations):
        value = float(start)
        length = float(duration)
        if value < 0 or length < 0:
            raise ValueError(f"negative token timing: start={value}, duration={length}")
        out.append(Token(text=str(text), start=value, end=value + length))
    return out


PUNCTUATION_ONLY = re.compile(r"^[^\w\s]+$", re.UNICODE)


def words_from_tokens(tokens: Iterable[Token]) -> list[Word]:
    """Group sherpa-onnx subword tokens into Subvision-compatible words.

    A token beginning with whitespace starts a new word. Punctuation-only
    tokens extend the previous word. Any other non-leading-space token is a
    continuation of the current word.
    """
    words: list[Word] = []
    for token in tokens:
        text = token.text
        stripped = text.strip()
        if not stripped:
            continue

        starts_word = bool(text) and text[0].isspace()
        is_punctuation = bool(PUNCTUATION_ONLY.fullmatch(stripped))
        if not words or (starts_word and not is_punctuation):
            words.append(
                Word(text=stripped, start=token.start, end=token.end)
            )
            continue

        current = words[-1]
        joined = current.text + stripped
        words[-1] = Word(
            text=joined,
            start=min(current.start, token.start),
            end=max(current.end, token.end),
        )
    return words


SENTENCE_END = re.compile(r"[.!?]$", re.UNICODE)


def segments_from_words(words: list[Word]) -> list[dict[str, object]]:
    """Group words into Segments on sentence-ending punctuation."""
    segments: list[dict[str, object]] = []
    chunk: list[Word] = []

    def flush() -> None:
        if not chunk:
            return
        segments.append(
            {
                "start": chunk[0].start,
                "end": chunk[-1].end,
                "text": " ".join(word.text for word in chunk),
                "words": [asdict(word) for word in chunk],
            }
        )
        chunk.clear()

    for word in words:
        chunk.append(word)
        if SENTENCE_END.search(word.text):
            flush()
    flush()
    return segments


def build_payload(result, model_dir: str) -> dict[str, object]:
    tokens = tokens_from_result(result)
    words = words_from_tokens(tokens)
    return {
        "text": str(result.text),
        "segments": segments_from_words(words),
        "tokens": [asdict(token) for token in tokens],
        "words": [asdict(word) for word in words],
        "meta": {
            "engine": "sherpa-onnx",
            "model_dir": os.path.abspath(model_dir),
            "language": "en",
        },
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("wav", help="16 kHz mono PCM s16le WAV file")
    parser.add_argument(
        "--model-dir",
        default=os.environ.get("PARAKEET_MODEL_DIR"),
        help="directory containing encoder/decoder/joiner/tokens (default: $PARAKEET_MODEL_DIR)",
    )
    parser.add_argument(
        "--threads",
        type=int,
        default=min(4, os.cpu_count() or 1),
        help="ONNX runtime thread count",
    )
    parser.add_argument(
        "--output",
        help="write JSON here; stdout when omitted",
    )
    parser.add_argument("--pretty", action="store_true", help="pretty-print JSON")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if not args.model_dir:
        raise SystemExit("missing --model-dir or $PARAKEET_MODEL_DIR")

    encoder = os.path.join(args.model_dir, "encoder.int8.onnx")
    decoder = os.path.join(args.model_dir, "decoder.int8.onnx")
    joiner = os.path.join(args.model_dir, "joiner.int8.onnx")
    tokens = os.path.join(args.model_dir, "tokens.txt")
    for path in (encoder, decoder, joiner, tokens):
        if not os.path.isfile(path):
            raise SystemExit(f"missing model file: {path}")

    samples, sample_rate = load_wav(args.wav)
    recognizer = sherpa_onnx.OfflineRecognizer.from_transducer(
        encoder=encoder,
        decoder=decoder,
        joiner=joiner,
        tokens=tokens,
        num_threads=max(1, args.threads),
        provider="cpu",
        debug=False,
        decoding_method="greedy_search",
        model_type="nemo_transducer",
    )
    stream = recognizer.create_stream()
    stream.accept_waveform(sample_rate, samples)
    recognizer.decode_stream(stream)
    payload = build_payload(stream.result, args.model_dir)

    encoded = json.dumps(
        payload,
        ensure_ascii=False,
        indent=2 if args.pretty else None,
        separators=None if args.pretty else (",", ":"),
    )
    if args.output:
        with open(args.output, "w", encoding="utf-8") as file:
            file.write(encoded)
            file.write("\n")
    else:
        print(encoded)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError) as exc:
        print(f"parakeet spike failed: {exc}", file=sys.stderr)
        raise
