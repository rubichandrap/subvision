# ADR-0010: Parakeet TDT v3 replaces whisper.cpp transcription

Date: 2026-11-04 · Status: accepted

## Context

Timing Drift survived every decoder-side adjustment recorded in ADR-0008.
whisper.cpp's word timestamps are largely interpolated between sparse token
anchors, so on real videos the first word could appear in silence and interior
words could run late. Recognition on produced audio was also poor with the
small `ggml-base.en` model.

A local Parakeet spike (issue #64, tools in `asr/spike`) produced native
token timings and materially better recognition. The user's private video
passed an ear/visual check locally: timing was nearly exact and a few words
were wrong, versus badly garbled text and drift on the whisper path. The
private video stays local.

Measured on the redistributable clean-speech fixture:

- Parakeet matched the hand transcript (WER 0.000).
- Against the whisper baseline on the same fixture, whisper word-start times
  had MAE 0.686 s, p95 2.770 s, worst 3.040 s, and a +0.674 s late bias.
  That number measures disagreement with Parakeet, not absolute ground truth.
- The music/silence/cut behaviour remains qualitative and local; no private
  media or transcript is committed.

## Decision

- **Engine:** NVIDIA Parakeet TDT 0.6B v3, int8 ONNX export, served through
  sherpa-onnx.
- **Worker boundary:** a small Python HTTP worker owns inference. The Go server
  posts the extracted 16 kHz mono WAV and receives Subvision's segment JSON.
  `PARAKEET_URL` is required; the server fails at boot without it.
- **No fallback:** worker unavailable, timeout, malformed JSON, or missing
  timestamps fail the Process loudly. whisper.cpp is not a fallback.
- **One production path:** whisper.cpp and its Go/cgo wiring are removed.
- **Reflow:** the server groups Timed Words into cue-sized segments after trim
  shifting and before persistence. Each cue is one Segment and carries at most
  `captions.wordsPerPage` words.
- **Empty transcript:** an empty transcript is a successful result and renders a
  clean video with no captions (ADR-0008 remains useful only as history).
- **Language:** English only in this cut. Translation remains separate.
- **Model distribution:** weights are pinned and mounted/baked externally; they
  are not committed and are never downloaded at response time. Pinned file
  hashes and license attribution live in `asr/worker/README.md` and
  `THIRD_PARTY_NOTICES.md`.

## Alternatives considered

- **More whisper decoder tuning / DTW / VAD** — ADR-0007 and ADR-0008 already
  exhausted the stock binding without moving the reporter's drift.
- **WhisperX or a forced aligner** — could improve word boundaries with a second
  model, but leaves recognition quality and a larger dependency surface. Not
  attempted automatically after the Parakeet spike won.
- **Commercial ASR APIs** — better recognition possible, but adds cost, privacy,
  and availability dependencies to a self-hosted app.
- **Lyrics-catalog forced alignment** — strongest for known songs but is a
  different product feature with licensing implications, not a general ASR fix.

## Consequences

- `WHISPER_MODEL_PATH`, the whisper submodule, the cgo build steps, and the
  whisper-specific fixtures/commands are retired from the production path.
- Deployments need a Parakeet worker container and roughly 640 MB of model
  files; CPU inference is acceptable but slower than whisper `base.en`.
- The model is CC-BY-4.0 and requires attribution.
- Cue reflow now owns subtitle line grouping; ADR-0005's transcribe-time
  whisper split behavior is replaced by server-side reflow.
- Re-rendering and transcript edits keep native Timed Words; same-token-count
  text edits update word text, while word-count changes are rejected loudly.
