# VAD + DTW word timestamps: captions timed to real speech onset (spec)

**GitHub issue:** #25 (labelled `ready-for-agent`)

**Status:** open — specified after a grilling session (2026-09-06); ADR-0007
landed first, implementation deliberately deferred until this spec existed.
**Revised 2026-09-06 (#28):** the owner ruled vendored third-party code is
never edited. #27's vendored binding is reverted, DTW is dropped (#28
wontfix), and VAD gating is rebuilt in the transcriber's own code — ffmpeg
silencedetect + windowed decode over the stock binding (ADR-0007, second
amendment). Sections below reflect the revision.

## Problem Statement

Word-driven captions (the pop animation above all) sometimes show a word before
it is spoken — seen both at video start and mid-video. The caption onset gate
(ADR-0006) is verified sound: a caption never mounts before its first reported
Timed Word. The timestamps themselves are the problem — whisper's token
timestamps are estimates, and the vendored build stamps the first tokens inside
leading silence (on the acceptance fixture, "And" is reported at 0.00 s while
speech is measured at 3.10 s). Mid-video, music and long pauses invite
hallucinated words, and within continuous speech the default heuristic
timestamps drift. No caption-side code can fix wrong timestamps: the fix
belongs upstream, in how transcription produces Timed Words.

## Solution

Enable word-timestamp-faithful transcription by keeping silence and music
out of the decoder, using only stock third-party code:

- **Speech gating (revised)**: the transcriber runs ffmpeg `silencedetect`
  over the converted wav and decodes only the detected speech windows
  (stock binding's `SetOffset`/`SetDuration`; the audio buffer is never
  cut, so reported timings stay on the real timeline). Leading-silence
  stamping and mid-video hallucinations disappear at the source. No
  vendored edits, no VAD model download.
- ~~**DTW token timestamps**~~: dropped (#28 wontfix) — DTW can only be
  configured through `whisper_context_params` at model load, which the
  stock Go binding does not expose, and vendored edits are off the table.
  Within-speech heuristic drift is accepted.

With trustworthy Timed Words, the existing onset gate makes every word-driven
caption — pop, karaoke, and the gated fade/slide entrances — appear with the
voice. VAD-style gating is adopted by configuration (a `VAD_GATING` env
flag). See ADR-0007 (second amendment).

## User Stories

1. As a viewer, I want a pop caption to appear only when its word is spoken, so
   that text never precedes the voice.
2. As a viewer, I want no caption during a long leading silence, so that a
   quiet intro never pulls text forward.
3. As a viewer, I want no caption during mid-video music or long pauses, so
   that hallucinated words never appear.
4. As a viewer, I want karaoke highlighting to track the voice, so that
   emphasis lands on the word being spoken.
5. As a viewer, I want fade captions to enter at the real voice onset, so that
   their entrance is not spent on silence.
6. As a viewer, I want slide captions to enter at the real voice onset, so that
   their entrance is not spent on silence.
7. As a viewer watching fast speech, I want each pop to land on its word's
   frame, so that rapid captions stay legible and in sync.
8. As an uploader of a video with a quiet intro, I want the first caption timed
   to the actual voice onset, so that the output looks professionally timed.
9. As a content creator posting short-form video, I want word timings locked to
   the voice, so that the pop style feels crisp rather than sloppy.
10. As an operator, I want VAD enabled by setting one env var to the Silero
    model path, so that adoption is a configuration change.
11. As an operator, I want an unset VAD path to preserve today's behavior, so
    that rollout and rollback are one-line changes.
12. As an operator, I want a configured-but-broken VAD path to fail
    transcription loudly, so that misconfiguration can never silently degrade
    output.
13. As an operator, I want a video with no speech to transcribe to zero
    segments with a warning, so that a caption-less output is understood as
    correct, not broken.
14. As an operator, I want DTW enabled automatically with the preset matching
    the loaded model, so that timestamp fidelity needs no manual tuning.
15. As an operator swapping whisper models, I want an unrecognized model to
    fall back to heuristic timestamps with a warning, so that transcription
    keeps working without DTW.
16. As an operator verifying the fix, I want the onset acceptance fixture
    regenerated with VAD+DTW and its measured numbers recorded, so that
    acceptance is measured against real audio, not eyeballed.
17. As an operator accepting the feature, I want the final gate to be an ear
    check on the reporter's original videos, so that "fixed" means what a
    viewer actually hears.
18. As a developer, I want VAD and DTW wiring tested at the transcriber seam
    with fakes, so that CI stays hermetic and fast.
19. As a developer, I want the vendored shim changes kept minimal and
    localized, so that a future vendor upgrade stays feasible.
20. As a developer, I want the caption onset gate and threshold constants
    untouched, so that ADR-0006's decisions remain standing on top of better
    timestamps.
21. As a developer, I want the new env var documented alongside the existing
    whisper model path, so that setup is discoverable.

## Implementation Decisions

- **Gating activation (revised)**: the server's env config gains an optional
  boolean `VAD_GATING`; unset means today's behavior. Set means gating
  required: ffmpeg silencedetect (−30 dB / 0.5 s, the measured thresholds)
  failing fails the transcription loudly, and zero detected speech yields
  empty segments plus a warning log. Windows split only at silences ≥ 2 s
  (shorter pauses stay inside windows — measured on the fixture, windows
  that end at every pause leak past their end into the zero-padded chunk
  tail) and windows < 0.25 s are dropped (whisper's own
  min_speech_duration_ms default).
- ~~**DTW activation**~~: dropped — no preset mapping, no model-load
  extension (ADR-0007, second amendment).
- ~~**Vendored binding extension**~~: reverted. The binding is consumed
  stock from the module proxy; the submodule pin returns to `d1f114da`.
  Third-party code is never edited in place (owner decision).
- **One producer of Timed Words**: the transcriber remains the only source of
  word timings; word timings are still never guessed (glossary invariant).
  Segment splitting and the caption onset gate are untouched — they consume
  whatever timestamps whisper reports, which gating keeps on the real
  timeline.
- **Docs**: the server env example gains `VAD_GATING` alongside the whisper
  model path.

## Testing Decisions

- A good test asserts external behavior — given configuration and inputs, what
  the transcriber decides and what reaches the decoder — never cgo internals.
- **Transcriber seam (primary, existing)**: unit tests with fakes at the
  new seams: silencedetect output parsing (pure), speech-window derivation
  (pure: clamping, the ≥ 2 s split rule, the 0.25 s minimum), the clamp of
  collected segments to their window (pure), the no-speech policy (pure),
  and the Transcribe wiring with a faked silence detector (gating off never
  runs detection; a silent wav transcribes empty without the model; a
  failing detection fails loudly). The existing threshold-wiring tests keep
  passing unchanged.
- **Fixture seam (acceptance, existing)**: the onset fixture regen command
  runs with `VAD_GATING=true` and records measured numbers — first reported
  word within ±100 ms of the measured voice onset (≈3.33 s) — into the
  fixture and issue #25. The vfx template fixture test keeps locking the
  onset gate on the regenerated segments. CI stays hermetic: no model
  downloads, no whisper inference, no ffmpeg subprocess in unit tests (the
  detection runner is a faked seam).
- **Vendored shim (no unit tests)**: validated end-to-end by the fixture regen,
  per house convention of not unit-testing vendored code.
- Prior art: the transcriber threshold tests, the #23 onset fixture and its
  regen command, and the vfx fixture template test.

## Out of Scope

- Any renderer/template change: pop, karaoke, fade, slide, page sizing,
  styling.
- Whisper model swap or upgrade; word-threshold retuning;
  max-initial-timestamp changes; VAD parameter tuning beyond upstream
  defaults.
- Automatic model-to-preset mapping for arbitrary models beyond the small
  named mapping (unknown → DTW off).
- Changes to segment splitting or the VFX Job contract.

## Further Notes

- Decision record: ADR-0007 (supersedes ADR-0006's no-VAD stance and corrects
  its premise — the vendored whisper.cpp ships VAD and DTW natively; only the
  Go binding shim lacked exposure). ADR-0006's onset gate and threshold
  constants stand.
- Evidence: on the acceptance fixture, whisper reported "And" at 0.00 s, "so"
  at 0.68 s, "my" at 1.13 s while measured speech onset is 3.10 s; where
  timestamps are real, alignment is ~60 ms. The reporter's symptom shape
  (start + mid-video) motivates enabling VAD and DTW together.
- Upstream surfaces this builds on: the `whisper_full_params` VAD fields
  (enable flag, model path, VAD params — with native timestamp remap inside
  `whisper_full`) and the context-params DTW fields with per-model
  alignment-head presets (a base.en preset exists).
- Fixture: `server/testdata/onset-fixture/` (regenerable via the onset fixture
  command); #20 holds the verification numbers this spec builds on.
