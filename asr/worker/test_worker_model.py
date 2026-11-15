"""Acceptance test against the real Parakeet model. Skipped without one.

It needs numpy, sherpa-onnx and the model files, so run it where the worker
runs, with the repository and models mounted (see "Tests" in README.md):

    docker run --rm --entrypoint python -w /repo/asr/worker \\
      -v "$PWD":/repo:ro -v "$PWD/models/parakeet":/models:ro \\
      -e PARAKEET_MODEL_DIR=/models subvision-asr-worker \\
      -m unittest discover -p 'test_*.py'
"""

from __future__ import annotations

import io
import os
import re
import unittest
import wave
from pathlib import Path

try:
    import main as worker_main
except ImportError:  # no numpy / sherpa-onnx on this machine
    worker_main = None

from chunking import ChunkOptions

REPO = Path(__file__).resolve().parents[2]
FIXTURE = REPO / "server/testdata/onset-fixture/fixture.wav"
REFERENCE = REPO / "asr/spike/reference.txt"
MODEL_DIR = os.environ.get("PARAKEET_MODEL_DIR", "")

READY = bool(
    worker_main is not None
    and MODEL_DIR
    and Path(MODEL_DIR, "encoder.int8.onnx").is_file()
    and FIXTURE.is_file()
    and REFERENCE.is_file()
)

# Chunks much smaller than the 24 s fixture put several joins inside its
# speech; moving where the audio starts moves every join across different words.
START_OFFSETS_SECONDS = (0.0, 0.37, 0.8, 1.3, 1.9, 2.6)
STRESS_SETTINGS = ((6, 2), (10, 4))


def normalize(text: str) -> str:
    return re.sub(r"[^\w']+", "", text.lower())


def word_error_rate(hypothesis: list[str], reference: list[str]) -> float:
    row = list(range(len(reference) + 1))
    for i, hyp in enumerate(hypothesis, 1):
        diagonal, row[0] = row[0], i
        for j, ref in enumerate(reference, 1):
            diagonal, row[j] = row[j], min(row[j] + 1, row[j - 1] + 1, diagonal + (hyp != ref))
    return row[len(reference)] / len(reference)


def without_the_first(seconds: float, wav: bytes) -> bytes:
    """The same 16 kHz mono WAV with its first ``seconds`` cut off."""
    with wave.open(io.BytesIO(wav)) as reader:
        reader.readframes(round(seconds * reader.getframerate()))
        rate, frames = reader.getframerate(), reader.readframes(reader.getnframes())
    out = io.BytesIO()
    with wave.open(out, "wb") as writer:
        writer.setnchannels(1)
        writer.setsampwidth(2)
        writer.setframerate(rate)
        writer.writeframes(frames)
    return out.getvalue()


@unittest.skipUnless(READY, "needs numpy, sherpa-onnx and PARAKEET_MODEL_DIR")
class RealModelTest(unittest.TestCase):
    workers: dict[tuple[float, float], object] = {}

    @classmethod
    def setUpClass(cls):
        cls.wav = FIXTURE.read_bytes()
        cls.reference = [normalize(w) for w in REFERENCE.read_text().split() if normalize(w)]

    def worker(self, chunk_seconds: float, overlap_seconds: float):
        key = (chunk_seconds, overlap_seconds)
        if key not in self.workers:  # loading the model is the slow part
            self.workers[key] = worker_main.ParakeetWorker(
                MODEL_DIR, threads=4, chunking=ChunkOptions(chunk_seconds, overlap_seconds)
            )
        return self.workers[key]

    def transcribe(self, chunk_seconds: float, overlap_seconds: float, wav: bytes | None = None) -> dict:
        return self.worker(chunk_seconds, overlap_seconds).transcribe_wav_bytes(wav or self.wav)

    def spoken(self, payload: dict) -> list[str]:
        return [normalize(w["text"]) for w in payload["words"] if normalize(w["text"])]

    def assert_one_clean_word_list(self, payload: dict) -> None:
        spoken = self.spoken(payload)
        self.assertEqual(len(spoken), len(self.reference), "a word was repeated or dropped")
        self.assertEqual([a for a, b in zip(spoken, spoken[1:]) if a == b], [])
        self.assertLessEqual(word_error_rate(spoken, self.reference), 0.1)
        words = payload["words"]
        for before, after in zip(words, words[1:]):
            self.assertLessEqual(before["end"], after["start"] + 1e-3)
        for word in words:
            self.assertLessEqual(word["start"], word["end"])

    def test_a_clip_that_fits_one_pass_is_transcribed_in_one_pass(self):
        payload = self.transcribe(30, 4)  # the fixture is 24 s long
        self.assertEqual(payload["meta"]["chunks"], 1)
        self.assertEqual(self.spoken(payload), self.reference)

    def test_the_default_settings_give_one_clean_word_list(self):
        model = worker_main.ParakeetWorker(MODEL_DIR, threads=4)
        self.assert_one_clean_word_list(model.transcribe_wav_bytes(self.wav))

    def test_chunks_smaller_than_the_speech_give_one_clean_word_list_wherever_the_audio_starts(self):
        for offset in START_OFFSETS_SECONDS:
            wav = without_the_first(offset, self.wav) if offset else self.wav
            for chunk, overlap in STRESS_SETTINGS:
                with self.subTest(start=offset, chunk=chunk, overlap=overlap):
                    payload = self.transcribe(chunk, overlap, wav)
                    self.assertGreater(payload["meta"]["chunks"], 1)
                    self.assert_one_clean_word_list(payload)

    def test_the_response_is_one_consistent_word_list(self):
        payload = self.transcribe(8, 3)
        self.assertNotIn("tokens", payload)
        self.assertEqual(payload["text"], " ".join(w["text"] for w in payload["words"]))
        in_segments = [w for s in payload["segments"] for w in s["words"]]
        self.assertEqual(in_segments, payload["words"])


if __name__ == "__main__":
    unittest.main()
