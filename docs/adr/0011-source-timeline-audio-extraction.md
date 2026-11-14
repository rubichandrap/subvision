# ADR-0011: Extract ASR audio on the Source Timeline

Date: 2026-11-14 · Status: accepted · Issue: #72

## Context

Timing Drift persisted on music videos after the move to Parakeet (ADR-0010), whose token timings are native. The cause was upstream of the model. Ingestion extracted audio with `ffmpeg [-ss start] -i video -vn … [-to end] out.wav`, which writes samples back to back and drops the audio stream's position on the container timeline. Reproduced on generated clips:

- An audio stream starting 0.727 s into the container produced a WAV whose first sound sat at 0.023 s, so every caption was 0.7 s early.
- A 1 s timestamp gap in the audio produced a 6.0 s WAV instead of 7.0 s, so every word after a gap was early and the error accumulated.
- Trim 5 → 8 s produced 7.99 s of audio: after an input `-ss`, an output `-to` acts as a duration.

Remuxed and downloaded music videos commonly carry late audio starts and timestamp gaps; the clean fixtures did not, so earlier timing work never saw this.

## Decision

The `sourcetimeline` server module owns extraction, the mapping back to source time, and clipping to the trim window. Extraction always uses:

- `-af aresample=async=1:first_pts=0`, which pads a late start and fills timestamp gaps with silence, so WAV time equals container time minus the trim start;
- `-t (end - start)` for a bounded window, never `-to`.

Segments returned by the ASR Worker are shifted into source time and clipped to the window before persistence: words starting at or after the window end are dropped, words running past it are cut back, and segments left without words are dropped.

## Consequences

- Removing or "simplifying" either flag reintroduces Timing Drift. The module's tests generate offset, gap and trim clips with ffmpeg and fail on the old arguments.
- Processes transcribed before this change keep their old timings; there is no backfill. Re-uploading re-transcribes.
- vfx still shifts and clips segments at render, because edited segments may fall outside the window.
- Server tests need `ffmpeg` on PATH for the module's tests; they skip without it.
