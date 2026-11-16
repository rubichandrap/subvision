"""Real-Demucs test for vocal isolation (ADR-0013). Skipped without torch and demucs.

It needs the image built with --build-arg VOCAL_ISOLATION=1, which also bakes in the
weights, so run it there with the repository mounted:

    docker run --rm --entrypoint python -w /repo/asr/worker -v "$PWD":/repo:ro \\
      subvision-asr -m unittest test_isolation_model
"""

from __future__ import annotations

import unittest
from pathlib import Path

try:
    import numpy as np
    import torch  # noqa: F401
    import demucs  # noqa: F401
    from isolation import apply_isolation, load_demucs_isolator
    from main import load_wav
except ImportError:  # default image: no torch / demucs
    load_demucs_isolator = None

FIXTURE = Path(__file__).resolve().parents[2] / "server/testdata/onset-fixture/fixture.wav"


@unittest.skipUnless(load_demucs_isolator and FIXTURE.is_file(), "Demucs or fixture not available")
class RealIsolation(unittest.TestCase):
    def test_vocals_keep_length_dtype_and_signal(self):
        samples, rate = load_wav(FIXTURE)
        isolate = load_demucs_isolator(2)
        out = apply_isolation(samples, rate, isolate)
        self.assertEqual(out.shape, samples.shape)
        self.assertEqual(out.dtype, np.float32)
        self.assertTrue(np.isfinite(out).all())
        # The fixture is clean speech, so the vocals stem must still carry it.
        self.assertGreater(float(np.sqrt((out**2).mean())), 0.2 * float(np.sqrt((samples**2).mean())))


if __name__ == "__main__":
    unittest.main()
