# ADR-0007: Speech gating with ffmpeg silencedetect (not a VAD model)

Date: 2026-09-06 · Status: SUPERSEDED (removed 2026-09-07 — failed XY experiment for Timing Drift, did not fix drift on the reporter's video; see the drop-speech-gating spec in `.scratch/drop-speech-gating/spec.md`. Every transcription decodes the whole audio in one pass; the ADR-0006 onset gate stands.)

The name history is confusing, so the fact first: no VAD model is used
anywhere. The mechanism is ffmpeg `silencedetect` (already a dependency)
finding the speech windows, and whisper decoding only those windows. An
earlier revision of this ADR proposed whisper.cpp's built-in VAD plus a
Silero model download; that was reverted — the vendored code is never edited
in place, and the stock Go binding cannot read the remapped token timestamps
built-in VAD needs. The `ggml-silero` binary briefly present in the models
folder was deleted as an orphan. The env flag was renamed from `VAD_GATING`
to `SPEECH_GATING` to say what it is; a stale `VAD_GATING` fails loudly
rather than silently turning gating off.

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

## Amendment (2026-09-06, issue #27)

Implementing VAD moved the vendored pin from `d1f114da` to the v1.9.3
release: at `d1f114da` whisper.cpp remaps segment timestamps under VAD but
not token timestamps — the token/word timings the transcriber builds on
stayed on the filtered (compressed) timeline — and its Go binding exposed no
VAD surface. v1.9.3 carries upstream's token-level remap (PR #3910) and VAD
binding setters; the decision above is unchanged. Measured on the acceptance
fixture, VAD's upstream defaults already track the voice onset — 3.33 s at
−30 dB silencedetect (the −35 dB crossing at 3.10 s is the sample's
room-tone ramp, not speech), first reported word 3.34 s — so no VAD
parameter moved. That measurement supersedes the −35 dB acceptance
reference above: the ±100 ms gate holds against the measured voice onset
(≈3.33 s), and it is that onset the fixture now records.

## Amendment (2026-09-06, issue #28 reverted; #27's vendoring reverted)

The owner ruled that vendored third-party code is never edited in place —
neither the whisper.cpp submodule nor the Go binding. That decision
unwinds this ADR's "extend the vendored shim" premise and forces two
mechanism changes:

- **The vendored Go binding is gone.** The server consumes
  `github.com/ggerganov/whisper.cpp/bindings/go` from the module proxy
  again and the submodule pin returns to `d1f114da` (both the pre-#27
  state). The binding's #27 extension (reading token times through the C
  remap getters) is withdrawn, which means whisper.cpp's *built-in* VAD can
  no longer be used: its remap exists only in the C getters
  (`whisper_full_get_token_t0/t1`), while the stock binding reads
  `TokenData.t0/t1`, which stay on the filtered (compressed) timeline —
  word timings would be seconds early. Measured in #27 and re-confirmed
  against the v1.9.3 and ggml-org sources.
- **DTW token timestamps are dropped** (#28 closed wontfix). The drift DTW
  would have shrunk — within-speech token timing error of a few hundred ms —
  is accepted. Within-speech alignment stays at heuristic quality; the
  onset gate (ADR-0006) stands on top of it.

VAD gating survives, rebuilt entirely in the transcriber's own code, with
the third-party surface stock end to end:

- `VAD_GATING` (optional, default off) enables gating. On: the transcriber
  runs ffmpeg `silencedetect` (−30 dB, 0.5 s minimum — the same measured
  thresholds as above) over the converted wav, derives the speech windows
  (splitting only at silences ≥ 2 s, dropping windows < 0.25 s), and decodes
  only those windows through the stock binding's `SetOffset`/`SetDuration`.
  The audio buffer is never cut, so whisper's reported timings stay on the
  original timeline — no C-side remap needed. Off: today's behavior, the
  whole audio in one pass, no ffmpeg subprocess.
- Fail-fast: with gating on, a silencedetect failure fails the
  transcription loudly; zero detected speech transcribes empty with a
  warning — the operator policies of the first amendment carry over.
- Measured on the acceptance fixture, gated decode reproduces the first
  amendment's numbers: first reported word 3.34 s against the measured
  ≈3.33 s onset, last word ends 13.52 s against a real ≈13.5 s. The
  window-split rule carries its own measurement: a 2 s window ending at a
  1 s pause leaked 2 s past its end (whisper decodes into the zero-padded
  tail of its mel chunk and re-reports the next window's words), so windows
  split only at pauses long enough to be skipped wholesale; collected words
  are clamped to their window as a second guard.

The upgrade path this ADR first imagined — upstream exposing the missing
surfaces and this repo consuming them stock — remains the route to closer
word timing: a PR to the Go binding reading token times through the remap
getters would re-enable whisper's built-in VAD without any vendoring.
