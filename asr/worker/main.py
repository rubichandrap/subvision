#!/usr/bin/env python3
"""Small HTTP worker that turns a 16 kHz mono WAV into Subvision Segments.

This is intentionally tiny and dependency-light: Subvision's Go server owns
ingestion, reflow, persistence, and rendering, while this worker owns only
Parakeet inference and token timing.
"""

from __future__ import annotations

import argparse
import contextlib
import json
import os
import re
import sys
import tempfile
import threading
import traceback
import wave
from dataclasses import asdict, dataclass
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Iterable

import numpy as np
import sherpa_onnx

# A WAV body is 16-bit mono at 16 kHz, so 256 MiB is roughly 2.3 hours. This
# is an internal service; keep a hard ceiling even if a caller lies.
MAX_BODY_BYTES = int(os.environ.get("ASR_MAX_BODY_BYTES", str(256 * 1024 * 1024)))
CONNECTION_TIMEOUT_SECONDS = 60


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


class ParakeetWorker:
    """Thread-safe wrapper around one sherpa-onnx recognizer."""

    def __init__(self, model_dir: str, threads: int = 4) -> None:
        encoder = Path(model_dir) / "encoder.int8.onnx"
        decoder = Path(model_dir) / "decoder.int8.onnx"
        joiner = Path(model_dir) / "joiner.int8.onnx"
        tokens = Path(model_dir) / "tokens.txt"
        missing = [path for path in (encoder, decoder, joiner, tokens) if not path.is_file()]
        if missing:
            raise FileNotFoundError(
                "missing Parakeet model files: " + ", ".join(str(path) for path in missing)
            )

        self._recognizer = sherpa_onnx.OfflineRecognizer.from_transducer(
            encoder=str(encoder),
            decoder=str(decoder),
            joiner=str(joiner),
            tokens=str(tokens),
            num_threads=max(1, threads),
            provider="cpu",
            debug=False,
            decoding_method="greedy_search",
            model_type="nemo_transducer",
        )
        self._model_dir = str(Path(model_dir).resolve())
        self._lock = threading.Lock()

    def transcribe_wav_bytes(self, data: bytes) -> dict[str, object]:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "audio.wav"
            path.write_bytes(data)
            samples, sample_rate = load_wav(path)
            with self._lock:
                stream = self._recognizer.create_stream()
                stream.accept_waveform(sample_rate, samples)
                self._recognizer.decode_stream(stream)
                result = stream.result
            tokens = tokens_from_result(result)
            words = words_from_tokens(tokens)
            return {
                "text": str(result.text),
                "segments": segments_from_words(words),
                "tokens": [asdict(token) for token in tokens],
                "words": [asdict(word) for word in words],
                "meta": {
                    "engine": "sherpa-onnx",
                    "language": "en",
                },
            }


def load_wav(path: Path) -> tuple[np.ndarray, int]:
    with contextlib.closing(wave.open(str(path), "rb")) as reader:
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
    """Group sherpa-onnx subword tokens into Subvision-compatible words."""
    words: list[Word] = []
    for token in tokens:
        text = token.text
        stripped = text.strip()
        if not stripped:
            continue

        starts_word = bool(text) and text[0].isspace()
        is_punctuation = bool(PUNCTUATION_ONLY.fullmatch(stripped))
        if not words or (starts_word and not is_punctuation):
            words.append(Word(text=stripped, start=token.start, end=token.end))
            continue

        current = words[-1]
        words[-1] = Word(
            text=current.text + stripped,
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


class Handler(BaseHTTPRequestHandler):
    worker: ParakeetWorker

    def setup(self) -> None:
        super().setup()
        self.connection.settimeout(CONNECTION_TIMEOUT_SECONDS)

    def do_GET(self) -> None:  # noqa: N802
        if self.path == "/healthz":
            self._send_json({"status": "ok"})
            return
        self._send_json({"error": "not found"}, status=404)

    def do_POST(self) -> None:  # noqa: N802
        if self.path != "/transcribe":
            self._send_json({"error": "not found"}, status=404)
            return

        content_type = self.headers.get("Content-Type", "").split(";")[0].strip()
        if content_type != "audio/wav":
            self._send_json(
                {"error": f"expected Content-Type audio/wav, got {content_type or 'missing'}"},
                status=400,
            )
            return

        length = self.headers.get("Content-Length")
        if not length or not length.isdigit():
            self._send_json({"error": "missing or invalid Content-Length"}, status=400)
            return

        size = int(length)
        if size <= 0:
            self._send_json({"error": "empty request body"}, status=400)
            return
        if size > MAX_BODY_BYTES:
            self._send_json(
                {"error": f"request body exceeds {MAX_BODY_BYTES} bytes"},
                status=413,
            )
            return

        data = self.rfile.read(size)
        try:
            payload = self.worker.transcribe_wav_bytes(data)
        except (ValueError, OSError, RuntimeError):
            # Log the concrete cause locally; don't return filesystem paths or
            # model details to the caller.
            print("[asr-worker] transcription failed:", file=sys.stderr)
            traceback.print_exc(file=sys.stderr)
            self._send_json({"error": "transcription failed"}, status=500)
            return
        self._send_json(payload)

    def _send_json(self, payload: dict[str, object], status: int = 200) -> None:
        body = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format: str, *args) -> None:  # noqa: A002
        sys.stderr.write("[asr-worker] " + (format % args) + "\n")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--model-dir",
        default=os.environ.get("PARAKEET_MODEL_DIR"),
        help="directory containing encoder/decoder/joiner/tokens (default: $PARAKEET_MODEL_DIR)",
    )
    parser.add_argument("--host", default=os.environ.get("ASR_HOST", "0.0.0.0"))
    parser.add_argument("--port", type=int, default=int(os.environ.get("ASR_PORT", "8081")))
    parser.add_argument(
        "--threads",
        type=int,
        default=int(os.environ.get("ASR_THREADS", "4")),
        help="ONNX runtime thread count",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if not args.model_dir:
        raise SystemExit("missing --model-dir or $PARAKEET_MODEL_DIR")

    worker = ParakeetWorker(args.model_dir, threads=args.threads)
    Handler.worker = worker
    server = ThreadingHTTPServer((args.host, args.port), Handler)
    sys.stderr.write(f"[asr-worker] listening on {args.host}:{args.port}\n")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        server.server_close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
