# 27 — VAD-gated transcription via VAD_MODEL_PATH

**What to build:** An operator who sets the VAD model path gets VAD-gated
transcription: the Silero VAD model detects the speech regions, whisper
transcribes only speech and reports its timings on the real timeline — so first
words no longer carry clamped leading-silence stamps and mid-video music or
pauses produce no hallucinated words. Unset means today's behavior. Set means
VAD required: an unloadable VAD model fails the transcription visibly (fail
fast), and zero detected speech transcribes empty with a warning log. VAD
parameters stay at whisper.cpp's upstream defaults. The vendored Go binding's
cgo shim is extended in place to carry the VAD enable flag, model path, and
default params into the decoder — no new binding, no fork.

**Blocked by:** #26 (threaded settings + pure policy seams).

**Status:** superseded 2026-09-06 — the vendored binding and the whisper.cpp
bump this ticket shipped are reverted (owner decision: third-party code
stays stock). VAD gating survives, rebuilt in the transcriber's own code
(ffmpeg silencedetect + windowed decode) behind `VAD_GATING`; see ADR-0007's
second amendment and the regenerated onset fixture.

- [ ] Setting the VAD model path env (named to mirror the whisper model path
      var) makes the decoder run with VAD enabled against the configured
      Silero model; the vendored shim change is minimal and localized.
- [ ] Unset env preserves today's behavior exactly — no VAD fields touched.
- [ ] Configured-but-unloadable VAD model fails transcription with a clear
      error; zero detected speech yields empty segments plus a warning log.
- [ ] Unit tests at the transcriber seam with fakes: a configured VAD path
      reaches the decoder context, an unset one leaves it untouched, and the
      policy decisions behave as specified.
- [ ] The onset fixture regen command runs with VAD enabled; the regenerated
      segments are recorded; the first reported word is within ±100 ms of the
      measured speech onset (≈3.10 s); the vfx fixture suite is green on the
      regenerated segments.
- [ ] Measured numbers posted as evidence on the parent issue.
