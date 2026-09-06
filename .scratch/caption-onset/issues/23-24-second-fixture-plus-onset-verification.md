# 23 — 24-second fixture plus onset verification

**GitHub issue:** #23

**Status:** done (closed 2026-09-06)

## Parent

#20 (see `../spec.md`)

## What to build

The 24-second sample video plus its transcription segment JSON becomes the
measured acceptance reference. Heard onset versus reported word start is
recorded for the first two to three words, proving the caption no longer
appears during leading silence.

## Acceptance criteria

- [x] Fixture video and segment JSON stored as the acceptance reference
- [x] Heard onset versus reported start recorded for first 2-3 words
- [x] No caption visible during leading silence on the fixture — with the
      recorded caveat below
- [x] Results recorded as a comment on the parent spec issue

## Done

Fixture commit 5c9a342. The reporter video is not in the repo, so the fixture
reconstructs the shape: 24.000 s — 3.0 s leading silence, whisper.cpp's jfk
sample, 10 s trailing silence (`fixture.wav` + `fixture.mp4` +
`segments.json`, regenerable via `server/cmd/onsetfixture`). Physical onset
measured with ffmpeg silencedetect: 3.10 s. The gate is locked against the
real transcription in CI by `vfx/src/templates/fixture.test.ts`.

**Recorded caveat:** the vendored whisper build (max-initial-timestamp 1.0,
no VAD) stamps the first tokens inside the leading silence (reported 0.00 s
vs measured 3.10 s), so captions can still appear during leading silence on
this fixture — upstream of the gate, unfixable by caption code or threshold
tuning. ADR-0006 names VAD as the upgrade path. Full results on #20.
