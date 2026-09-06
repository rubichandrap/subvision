# VAD + DTW word timestamps: captions timed to real speech onset (spec)

**GitHub issue:** #25 (labelled `ready-for-agent`)

**Status:** open — specified after a grilling session (2026-09-06); ADR-0007
landed first, implementation deliberately deferred until this spec existed.

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

Enable the two word-timestamp mechanisms the vendored whisper.cpp already
ships, both exposed through a small extension of the vendored Go binding's cgo
shim:

- **VAD** (Voice Activity Detection, Silero): before decoding, silence and
  music are detected and dropped, so whisper only ever transcribes real speech
  and reports its timings on the real timeline. Leading-silence stamping and
  mid-video hallucinations disappear at the source.
- **DTW** token timestamps: whisper's per-token time estimates are computed by
  aligning tokens to audio frames (dynamic time warping) instead of the coarse
  default heuristic, so within-speech drift shrinks.

With trustworthy Timed Words, the existing onset gate makes every word-driven
caption — pop, karaoke, and the gated fade/slide entrances — appear with the
voice. VAD is adopted by configuration (an optional VAD model path); DTW turns
on with the model. See ADR-0007.

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

- **VAD activation (ADR-0007)**: the server's env config gains an optional
  `VAD_MODEL_PATH` mirroring `WHISPER_MODEL_PATH`; the Silero model is fetched
  with the vendored download script into the models directory the compose file
  already mounts. Unset means today's behavior. Set means VAD required: the
  transcription fails if the VAD model cannot be loaded (fail fast — the job
  fails visibly), and zero detected speech yields empty segments plus a warning
  log. VAD parameters stay at whisper.cpp's upstream defaults; they move only
  on measured evidence, like the threshold constants.
- **DTW activation (ADR-0007)**: DTW token timestamps are enabled at model load
  with the alignment-heads preset matching the loaded model; the current model
  maps to the base.en preset. The preset mapping lives on the transcriber side;
  an unrecognized model means DTW disabled plus a warning (never a failure).
  DTW is not separately gated.
- **Vendored binding extension**: the vendored Go binding's cgo shim is
  extended in place — the full-params wrapper carries VAD (enable flag, model
  path, upstream-default params) and the model-load path carries DTW (enable
  flag, preset). No new binding, no fork of whisper.cpp.
- **One producer of Timed Words**: the transcriber remains the only source of
  word timings; word timings are still never guessed (glossary invariant).
  Segment splitting and the caption onset gate are untouched — they consume
  whatever timestamps whisper reports, which this feature makes real.
- **Docs**: the server env example and README whisper setup gain the VAD model
  path and its download step.

## Testing Decisions

- A good test asserts external behavior — given configuration and inputs, what
  the transcriber decides and what reaches the decoder — never cgo internals.
- **Transcriber seam (primary, existing)**: extend the transcriber unit tests
  with fakes for the new decoder-context surface: a configured VAD path reaches
  the context and an unset one does not; the DTW preset resolves for the known
  model and disables with a warning for an unknown one; the fail-fast decision
  is a pure function. The existing threshold-wiring tests (fake setter pattern)
  keep passing unchanged.
- **Fixture seam (acceptance, existing)**: the onset fixture regen command runs
  with VAD+DTW enabled and records measured numbers — first reported word
  within ±100 ms of the measured speech onset (≈3.10 s by silencedetect
  −35 dB / 0.5 s) — into the fixture and issue #25. The vfx template fixture
  test keeps locking the onset gate on the regenerated segments. CI stays
  hermetic: no model downloads, no whisper inference in CI.
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
