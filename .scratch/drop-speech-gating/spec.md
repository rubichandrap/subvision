Promoted: #31

# Spec: Drop SPEECH_GATING (failed XY experiment)

## Problem Statement

Speech gating (ffmpeg `silencedetect` finding speech windows, decoding only
those) was built to fix Timing Drift: silence pulling words forward and
hallucinations over music. It did not fix drift on the reporter's video, and
the remaining decoder knobs (beam, entropy, larger model) failed too
(ADR-0008). The feature now carries cost without benefit: an extra ffmpeg
subprocess per transcription, a second decode path to maintain, an env flag,
fixture variants, and a misleading history (built-in VAD proposed, reverted;
flag renamed from VAD_GATING). It is a classic XY solution: built for drift,
kept for onset, but onset is already handled by the ADR-0006 gate.

## Solution

Remove speech gating entirely. Every transcription decodes the whole audio in
one pass — the pre-gating behavior that is already the default. Delete the
flag, the window-splitting code, the clamping, the fixture-regen path, and
all docs describing gating as a feature.

## User Stories

1. As a video owner, I want transcription to behave one way, so that I never wonder which path my video took.
2. As a video owner, I want no gating-related failure modes (silence detection errors), so that one fewer thing can break my upload.
3. As a deployer, I want one fewer env flag to set, so that configuration stays minimal.
4. As a maintainer, I want one decode path, so that future timing work has a single place to change.
5. As a maintainer, I want the onset gate (ADR-0006) kept intact, so that leading-silence captions stay fixed after gating is gone.

## Implementation Decisions

- Delete the `SPEECH_GATING` env flag and its loud-failure guard for the old
  `VAD_GATING` name, the `SpeechGating` settings field, the silence-interval
  parsing, speech-window derivation, per-window decode loop, and window
  clamping. `Transcribe` decodes the full audio buffer in one context.
- The onset gate (ADR-0006) and word timestamp thresholds stay exactly as
  they are — they are independent of gating and verified by the fixture.
- The acceptance fixture keeps its current recorded segments (transcribed
  gated); it is re-recorded once, ungated, with the regen command, and the
  measured onset numbers are updated if they move.
- Docs: ADR-0007 is marked superseded with a pointer to this removal; the
  README speech-gating section and env table row are deleted; the
  `SPEECH_GATING` glossary entry in CONTEXT.md is removed; the fixture
  regen comment drops the gating variant.
- The `server/.env` on the operator machine may still carry
  `SPEECH_GATING=true`; unknown env vars are ignored, so no operator action
  is required. (The loud-failure guard being deleted only rejects the old
  `VAD_GATING` name, which also becomes inert.)

## Testing Decisions

- Test external behavior only: full-audio decode in one pass, onset gate
  still holding on the re-recorded fixture, no silence-detection subprocess
  invoked.
- Delete the gating unit tests (silence parsing, window derivation,
  no-speech wiring); keep and re-point the onset fixture test at ungated
  segments.
- Prior art: the existing transcriber wiring tests and the vfx onset
  fixture test.

## Out of Scope

- Any replacement silence handling. One-pass decode plus the onset gate is
  the whole story.
- Reverting ADR-0006. The onset gate stays.
- Touching the vfx side. Render never knew about gating.

## Further Notes

- User verdict 2026-09-07: gating is an XY solution for Timing Drift that
  did not work; clean it up rather than keep it as a half-feature.
- If drift work ever needs windowed decode again, it returns as a new
  proposal with fresh measurements, not as a resurrection of this flag.
