# ADR-0007: Enable VAD and DTW word timestamps through the vendored binding

Date: 2026-09-06 · Status: accepted

ADR-0006 left VAD as the upgrade path believing it would mean a new cgo binding plus a model download. That premise was wrong: the vendored whisper.cpp (`d1f114da`) implements VAD natively — `whisper_full_params.vad` with `vad_model_path`, and `whisper_full` remaps decoded timestamps back onto the real timeline (`src/whisper.cpp:7804`) — and DTW token timestamps too (`whisper_context_params.dtw_token_timestamps`, with `WHISPER_AHEADS_BASE_EN` matching the `ggml-base.en` model we run). Only the vendored Go binding's cgo shim does not expose them, and extending that shim is a small change, not a new binding. With the onset gate verified sound but the reporter still seeing text before speech (at video start and mid-video), and the fixture proving whisper stamps first tokens inside leading silence ("And" reported 0.00 s vs measured 3.10 s — no caption-side code can fix that), we enable both mechanisms at once: VAD so silence and music never reach the decoder (killing clamped first tokens and mid-video hallucinations), and DTW so token timestamps align tightly to the frames that produced them (shrinking mid-speech drift).

## Decisions

- VAD is activated by an optional `VAD_MODEL_PATH` mirroring `WHISPER_MODEL_PATH`; the Silero model is fetched with the vendored `models/download-vad-model.sh` into `bindings/go/models/`, which compose already mounts. Unset means today's behavior; set means VAD required.
- A configured VAD model that fails to load fails fast rather than silently degrading; VAD detecting zero speech transcribes empty with a warning log (a video with no speech needs no captions).
- DTW is enabled at model load with the alignment-heads preset matching the loaded model, not separately gated.
- Word timestamp thresholds stay at the upstream defaults (ADR-0006) and the transcriber model stays `ggml-base.en`.

## Consequences

- Word timings come from whisper over speech-only audio: leading-silence clamping can no longer stamp tokens into silence, so caption onset follows the real voice onset while the ADR-0006 gate stays unchanged on top of better timestamps.
- DTW raises decode CPU and memory cost and is not separately gated; reversing it means removing the shim extension.
- ADR-0006's no-VAD stance and its new-binding premise are superseded; its onset gate and threshold constants stand.
- Acceptance: the onset fixture regenerates with VAD+DTW enabled and records measured numbers (first word ≈ 3.10 s ± 100 ms); CI stays hermetic (recorded fixture JSON plus the regen command); the reporter's ear check on the original videos remains the final gate.
