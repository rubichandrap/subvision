"""Acceptance test against the real Parakeet model. Skipped without one.

It needs numpy, sherpa-onnx and the model files, so run it where the worker
runs, with the repository and models mounted:

    docker run --rm --entrypoint python \\
      -v "$PWD":/repo:ro -v "$PWD/models/parakeet":/models:ro \\
      -e PARAKEET_MODEL_DIR=/models subvision-asr \\
      -m unittest discover -s /repo/asr/worker -p 'test_*.py'
"""

from __future__ import annotations

import os
import re
import unittest
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

READY = (
    worker_main is not None
    and MODEL_DIR
    and Path(MODEL_DIR, "encoder.int8.onnx").is_file()
    and FIXTURE.is_file()
    and REFERENCE.is_file()
)


def normalize(text: str) -> str:
    return re.sub(r"[^\w']+", "", text.lower())


def word_error_rate(hypothesis: list[str], reference: list[str]) -> float:
    row = list(range(len(reference) + 1))
    for i, hyp in enumerate(hypothesis, 1):
        diagonal, row[0] = row[0], i
        for j, ref in enumerate(reference, 1):
            diagonal, row[j] = row[j], min(row[j] + 1, row[j - 1] + 1, diagonal + (hyp != ref))
    return row[len(reference)] / len(reference)


@unittest.skipUnless(READY, "needs numpy, sherpa-onnx and PARAKEET_MODEL_DIR")
class RealModelTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.wav = FIXTURE.read_bytes()
        cls.reference = [normalize(w) for w in REFERENCE.read_text().split() if normalize(w)]

    def transcribe(self, chunk_seconds: float, overlap_seconds: float) -> dict:
        model = worker_main.ParakeetWorker(
            MODEL_DIR, threads=4, chunking=ChunkOptions(chunk_seconds, overlap_seconds)
        )
        return model.transcribe_wav_bytes(self.wav)

    def spoken(self, payload: dict) -> list[str]:
        return [normalize(w["text"]) for w in payload["words"] if normalize(w["text"])]

    def assert_one_clean_transcript(self, payload: dict) -> None:
        spoken = self.spoken(payload)
        self.assertEqual(len(spoken), len(self.reference), "a word was repeated or dropped")
        self.assertEqual([a for a, b in zip(spoken, spoken[1:]) if a == b], [])
        self.assertLessEqual(word_error_rate(spoken, self.reference), 0.1)
        words = payload["words"]
        for before, after in zip(words, words[1:]):
            self.assertLessEqual(before["end"], after["start"] + 1e-3)

    def test_a_clip_that_fits_one_pass_is_transcribed_in_one_pass(self):
        payload = self.transcribe(30, 4)  # the fixture is 24 s long
        self.assertEqual(payload["meta"]["chunks"], 1)
        self.assertEqual(self.spoken(payload), self.reference)

    def test_the_default_settings_give_one_clean_transcript(self):
        model = worker_main.ParakeetWorker(MODEL_DIR, threads=4)
        self.assert_one_clean_transcript(model.transcribe_wav_bytes(self.wav))

    def test_chunks_smaller_than_the_speech_still_give_one_clean_transcript(self):
        payload = self.transcribe(8, 3)
        self.assertGreater(payload["meta"]["chunks"], 1)
        self.assert_one_clean_transcript(payload)

    def test_the_response_is_one_consistent_transcript(self):
        payload = self.transcribe(8, 3)
        self.assertNotIn("tokens", payload)
        self.assertEqual(payload["text"], " ".join(w["text"] for w in payload["words"]))
        in_segments = [w for s in payload["segments"] for w in s["words"]]
        self.assertEqual(in_segments, payload["words"])


if __name__ == "__main__":
    unittest.main()
