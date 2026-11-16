"""Optional vocal isolation before decoding (ADR-0013, experiment).

When ``ASR_VOCAL_ISOLATION`` is on, the worker separates the vocals from the
accompaniment with Demucs (``htdemucs``) and decodes the vocals stem. Off by
default: ADR-0013 measured no gain on one song. ``torch`` and ``demucs`` are
imported only when isolation is switched on, so the default image needs
neither.
"""

from __future__ import annotations

import os
from typing import Callable

import numpy as np

DEMUCS_MODEL = "htdemucs"
DEMUCS_RATE = 44100

Isolator = Callable[[np.ndarray, int], np.ndarray]


def enabled_from_env(raw: str | None) -> bool:
    """Parse ``ASR_VOCAL_ISOLATION``: 1/true/yes/on turn it on; unset or 0/false/no/off keep it off."""
    value = (raw or "").strip().lower()
    if value in ("", "0", "false", "no", "off"):
        return False
    if value in ("1", "true", "yes", "on"):
        return True
    raise ValueError(f"ASR_VOCAL_ISOLATION must be 0 or 1, got {raw!r}")


def apply_isolation(samples: np.ndarray, sample_rate: int, isolate: Isolator) -> np.ndarray:
    """Run ``isolate`` and check it hands back audio of the same length, so the
    word times still sit on the Source Timeline."""
    out = isolate(samples, sample_rate)
    if out.shape != samples.shape:
        raise ValueError(
            f"vocal isolation changed the audio length: {len(samples)} -> {len(out)} samples"
        )
    return out.astype(np.float32, copy=False)


def load_demucs_isolator(threads: int) -> Isolator:
    """Load htdemucs once and return a function: 16 kHz mono in, 16 kHz mono vocals out."""
    try:
        import torch
        import torchaudio.functional as F
        from demucs.apply import apply_model
        from demucs.pretrained import get_model
    except ImportError as error:
        raise RuntimeError(
            "ASR_VOCAL_ISOLATION=1 but Demucs is not installed in this image; "
            "rebuild it with --build-arg VOCAL_ISOLATION=1 (compose: ASR_VOCAL_ISOLATION=1 in .env)"
        ) from error

    torch.set_num_threads(max(1, threads))
    model = get_model(DEMUCS_MODEL)
    model.eval()
    vocals_index = model.sources.index("vocals")

    def isolate(samples: np.ndarray, sample_rate: int) -> np.ndarray:
        wave = torch.from_numpy(np.ascontiguousarray(samples)).float()[None]
        wave = F.resample(wave, sample_rate, DEMUCS_RATE).repeat(2, 1)  # (2, n) stereo
        ref = wave.mean(0)
        wave = (wave - ref.mean()) / (ref.std() + 1e-8)
        with torch.no_grad():
            stems = apply_model(model, wave[None], device="cpu", split=True, progress=False)[0]
        vocals = stems[vocals_index] * (ref.std() + 1e-8) + ref.mean()
        mono = F.resample(vocals.mean(0, keepdim=True), DEMUCS_RATE, sample_rate)[0]
        out = mono.numpy()
        n = len(samples)
        return np.pad(out, (0, max(0, n - len(out))))[:n]

    return isolate
