# Silero VAD vs ffmpeg silencedetect for word-timestamp gating

Date: 2026-09-07. Status: analysis only, no implementation.
Context: Subvision transcribes with whisper.cpp (stock Go binding) and renders
word-timed captions. Speech gating via ffmpeg `silencedetect` was built, then
removed after it failed to move Timing Drift (see ADR-0007, ADR-0008,
`.scratch/drop-speech-gating/spec.md`). Question: reinstall gating with
Silero VAD instead?

## Verdict

Do not reinstall. Gating (either flavor) targets the wrong layer for this
repo's open problem. Reconsider only under the conditions at the end.

## How each works

### ffmpeg silencedetect: energy threshold, not speech detection

`silencedetect` logs a message when input volume stays at or below a noise
tolerance for a minimum duration. Options: `noise` (default -60 dB),
`duration` (default 2 s), `mono` for per-channel evaluation. It reports
`silence_start` / `silence_end` / `silence_duration` metadata. It knows
nothing about speech; anything loud (music, room tone ramp, handling noise)
counts as non-silence.

- Source: https://ffmpeg.org/ffmpeg-filters.html#silencedetect

Subvision's removed implementation ran `silencedetect` (-30 dB, 0.5 s minimum)
over the converted wav, derived speech windows (split only at silences >= 2 s,
drop windows < 0.25 s), and decoded only those windows via the stock binding's
`SetOffset`/`SetDuration`. The audio buffer was never cut, so reported timings
stayed on the original timeline.

- Source: `docs/adr/0007-vad-and-dtw-word-timestamps.md` (second amendment)

### Silero VAD: neural speech-probability per chunk

Silero VAD is a pre-trained neural voice activity detector (~260K params,
~2 MB JIT, MIT licensed) outputting a speech probability per audio chunk.
`get_speech_timestamps` converts probabilities to speech segments with tunable
knobs: `threshold` (default 0.5), `min_speech_duration_ms` (250),
`min_silence_duration_ms` (100), `speech_pad_ms` (30), plus max-duration
splitting and overlap handling. It supports 8 kHz and 16 kHz sample rates and
was trained on 6000+ languages.

- Sources: https://github.com/snakers4/silero-vad,
  https://github.com/snakers4/silero-vad/blob/master/src/silero_vad/utils_vad.py
  (`get_speech_timestamps` signature and docstring),
  https://github.com/snakers4/silero-vad/wiki/Version-history-and-Available-Models

### whisper.cpp native VAD support

whisper.cpp passes audio through the VAD model first, extracts only detected
speech segments, and decodes those. Upstream states this reduces the audio
whisper processes and can significantly speed up transcription. Supported
models are fetched via `models/download-vad-model.sh` (currently
`silero-v5.1.2`, `silero-v6.2.0` from `huggingface.co/ggml-org/whisper-vad`).
VAD behavior is tuned with `whisper_vad_params`: threshold, min speech/silence
durations, max speech duration, speech padding, samples overlap.

- Sources: https://github.com/ggml-org/whisper.cpp/blob/master/README.md
  ("Voice Activity Detection (VAD)" section),
  https://github.com/ggml-org/whisper.cpp/blob/master/include/whisper.h
  (`whisper_vad_params`, `whisper_full_params.vad` / `vad_model_path`),
  https://github.com/ggml-org/whisper.cpp/blob/master/models/download-vad-model.sh

Two API facts matter for word timestamps. Token times from
`whisper_full_get_token_t0/t1` are mapped back to the original audio timeline
when VAD is enabled (a token landing in removed silence snaps to the nearest
speech boundary), while raw `TokenData.t0/t1` stays in VAD-processed time.
Separately, `whisper_full_n_vad_segments` / `whisper_full_get_vad_segment_t0/t1`
expose the detected speech boundaries on the original timeline.

- Source: https://github.com/ggml-org/whisper.cpp/blob/master/include/whisper.h
  (comments on `whisper_full_get_token_t0/t1` and the VAD-segments accessors)

## Accuracy comparison (primary-source numbers)

Silero's wiki reports ROC-AUC on 31.25 ms segments across meeting, call, and
noisy-speech datasets. Multi-domain validation: Silero v5 0.96, v6 0.97, vs
WebRTC VAD 0.73 and an unnamed commercial VAD 0.93. The wiki notes threshold
should be tuned per dataset; 0.5 is the lazy default.

- Source: https://github.com/snakers4/silero-vad/wiki/Quality-Metrics

`silencedetect` has no accuracy numbers because it is not a classifier: a
single dB threshold cannot separate speech from loud non-speech. Local
evidence confirms the failure mode: on the acceptance fixture the -35 dB
crossing at 3.10 s was the sample's room-tone ramp, not speech; -30 dB tracked
the real voice onset (~3.33 s). Music passing the energy threshold was the
original mid-video hallucination concern.

- Sources: https://ffmpeg.org/ffmpeg-filters.html#silencedetect (threshold
  semantics), `docs/adr/0007-vad-and-dtw-word-timestamps.md` (first amendment
  measurement)

## Failure modes

silencedetect:

- Loud non-speech (music, applause, room-tone ramp) passes the gate, so the
  decoder still sees it and can still hallucinate. Gating never addressed
  hallucinations over music, only pure silence.
- Threshold is per-file fragile: too sensitive and room tone becomes "speech";
  too strict and quiet consonants/word edges get cut from windows.
- Windowed decode leaked: a window ending at a short pause decoded into the
  zero-padded tail of whisper's mel chunk and re-reported the next window's
  words, needing a clamp as second guard (local measurement in ADR-0007).

Silero VAD:

- Aggressive settings clip real speech: high threshold or long
  `min_speech_duration_ms` drops short words ("a", "I", "ya"); short padding
  clips plosives and fricatives at edges. This directly deletes or shortens
  Timed Words, which is worse than drift (ADR-0008 already rejects dropping
  low-confidence words for the same reason).
- Token remap snapping: with built-in VAD, tokens falling in removed gaps snap
  to the nearest speech boundary (whisper.h). Near cut points this distorts
  word edges instead of measuring them.
- Repo-specific blocker: the stock Go binding reads `TokenData.t0/t1`, which
  stays on the filtered (compressed) timeline under built-in VAD; the remap
  exists only in the C getters the binding does not expose. Using built-in VAD
  today would stamp word timings seconds early. Measured in issue #27 and
  re-confirmed against v1.9.3 sources (ADR-0007 second amendment). The owner
  also ruled vendored third-party code is never edited, so extending the shim
  in place is off the table.

## Latency / compute cost

Silero: ~830 us per 31.25 ms chunk (JIT) / ~207 us (ONNX) on one thread of a
Threadripper 3960X, i.e. 36x / 151x real-time; batching or GPU improves
further. Model is ~2 MB. Negligible next to a whisper decode, and upstream
notes VAD can net speed up transcription by shrinking decoder input.

- Sources: https://github.com/snakers4/silero-vad/wiki/Performance-Metrics,
  https://github.com/snakers4/silero-vad (Key Features),
  https://github.com/ggml-org/whisper.cpp/blob/master/README.md (VAD section)

silencedetect: one extra ffmpeg subprocess per transcription plus a second
decode path to maintain. Compute is trivial; the cost is maintenance and new
failure modes. The removal spec calls this out explicitly: "carries cost
without benefit."

- Source: `.scratch/drop-speech-gating/spec.md`

## Why silencedetect gave zero gain on per-word accuracy

1. The open problem is within-speech token timing error (a few hundred ms
   heuristic drift), not silence handling. Gating only changes which windows
   reach the decoder; it cannot move timestamps inside decoded speech. All
   four decoder knobs (beam 8, entropy 2.0, small.en, token-prob logging)
   likewise failed on the reporter's video (ADR-0008).
2. Leading-silence onset was already fixed by the ADR-0006 onset gate, so
   gating's one real win duplicated an existing fix.
3. silencedetect cannot tell music from speech, so mid-video hallucinations
   over music survived gating.

- Sources: `docs/adr/0008-decoder-tuning-for-timing-drift.md`,
  `.scratch/drop-speech-gating/spec.md`,
  `docs/adr/0006-caption-onset-gate-and-word-threshold.md`

## When Silero actually helps vs hurts word accuracy

Helps (hallucination suppression, not drift repair):

- Long silences / noise-only stretches decoded into phantom words: neural VAD
  excludes them where an energy gate cannot.
- Music or background chatter at speech-like energy: Silero's speech
  probability separates these far better than a dB threshold (ROC-AUC 0.96+
  vs WebRTC 0.73 on the multi-domain set).
- Skipping silence shrinks decoder input and can speed up transcription.

Hurts (accuracy and timing damage):

- Over-tuned threshold / min-duration drops short function words entirely.
- Tight padding clips word edges, shifting reported start/end times.
- Built-in VAD through the stock Go binding corrupts every word timestamp
  (filtered-timeline bug above); external pre-pass with
  `SetOffset`/`SetDuration` plus window clamping avoids it but reintroduces
  the leak guard the repo just deleted.
- Snapping of gap tokens to speech boundaries rewrites edge timings.

## Conditions to reinstall (all must hold)

1. The complaint is hallucinations over silence/music/noise or empty
   caption-less videos needing explicit no-speech handling, NOT within-speech
   Timing Drift. For drift, the sanctioned routes are render-side
   compensation or user-editable timing export (ADR-0008), not gating.
2. VAD runs as an external pre-pass in the transcriber's own code
   (`SetOffset`/`SetDuration`, buffer never cut, words clamped to windows),
   or upstream exposes the token-remap getters in the stock Go binding. No
   vendored edits, no built-in `params.vad` through the stock binding.
3. Measured on the acceptance fixture plus the reporter's ear check before
   and after, one knob at a time (threshold first, defaults otherwise:
   0.5 / 250 ms / 100 ms / 30 ms pad / 0.1 s overlap), with a revert if the
   fixture does not move. Follow the decoder-tuning discipline (ADR-0008).
4. Threshold tuned for the actual content domain, not left at lazy 0.5 for
   all uploads; short-word retention checked explicitly on the fixture.

Until then: one-pass decode plus the ADR-0006 onset gate stays the whole
story.
