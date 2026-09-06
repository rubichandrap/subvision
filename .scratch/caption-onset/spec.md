# Caption onset gate + word timestamp threshold (spec)

**GitHub issue:** #20

**Status:** closed 2026-09-06 — implementation and verification delivered (GitHub: #20 closed).

## Problem Statement

Word-driven subtitles (karaoke and pop animations) can appear on screen while
the audio is still silent. The caption page holding the first word mounts as
soon as its Transcription Segment window opens, even when the first Timed Word
starts later. On ear check against a 24-second video, the text is already
visible before any speech begins. Fade and slide look safe on that sample only
because their 0.4s / 0.35s fade-in starts from transparent; a longer leading
silence would show them early too.

## Solution

Gate every word-driven caption on the first word's start time so nothing
mounts during leading silence, and tune the word timestamp probability
threshold through the setters the Go binding already exposes. No model change,
no VAD. See ADR-0006.

## User Stories

1. As a viewer, I want no caption visible during leading silence, so that text
   appears when speech starts.
2. As a viewer, I want the karaoke caption to appear exactly when its first
   word is spoken, so that highlighting matches the voice.
3. As a viewer, I want the pop caption to appear exactly when its first word
   is spoken, so that words do not pop in early or mount an empty box.
4. As a viewer, I want fade captions to stay fully transparent until speech
   starts even with a long leading silence, so that short-sample safety
   becomes a guarantee.
5. As a viewer, I want slide captions to stay fully transparent until speech
   starts even with a long leading silence, so that short-sample safety
   becomes a guarantee.
6. As an uploader of a video with leading silence longer than one second, I
   want the first caption timed to the actual voice onset, so that quiet
   intros do not pull text forward.
7. As an editor reusing the caption page size control, I want the onset gate
   to work for any words-per-page value, so that the fix does not depend on
   page size.
8. As an operator, I want the word timestamp threshold to default to the
   upstream default, so that behavior is unchanged until measured evidence
   justifies tuning.
9. As an operator verifying the fix, I want the 24-second sample video plus
   its transcription segments as a fixture, so that onset can be measured
   rather than eyeballed.
10. As a developer, I want wordless segments to render exactly as before, so
    that music or sound markers are unaffected.

## Implementation Decisions

- The shared caption paging helper gains the onset rule: a segment with words
  yields no page before its first word starts. This is the single seam all
  word-driven animations route through, so karaoke and pop are fixed once,
  not per template.
- The karaoke template keeps rendering the page it is given with no extra
  per-word filter; the pop template keeps its existing per-word filter for
  stagger inside a visible page. Neither template implements its own onset
  logic.
- Fade and slide keep their existing fade curves but must also respect the
  onset rule, turning sample-only safety into a guarantee for long leading
  silence.
- The server transcriber names the word timestamp thresholds as constants
  defaulting to the upstream defaults, set through the existing Go binding
  setters for token threshold and token sum threshold.
- Threshold values move only on measured evidence from the fixture video
  (heard onset versus reported word start). No VAD, no max-initial-timestamp
  change, no model swap: the vendored Go binding exposes no VAD API and no
  max-initial-timestamp setter, so either would mean a new binding plus
  vendored changes, disproportionate to this fix.
- Split segments keep verbatim word timestamps; unsplit segments pass through
  untouched. The onset gate works on whatever timestamps arrive, so it also
  covers the upstream initial-timestamp clamp behavior.

## Testing Decisions

- Test external behavior only: given a segment and a playback time, which
  caption words are visible. Do not test internal paging indexes.
- Render seam: extend the existing shared template test suite with pre-onset
  cases (time before first word start yields no words), onset edge (exactly
  at first word start yields the first page), and wordless-segment
  passthrough.
- Transcriber seam: extend the existing transcriber test suite with threshold
  wiring (defaults equal upstream; custom values reach the decoder context)
  and the existing missing-timestamp error path.
- Fixture: the 24-second sample video plus its segment JSON becomes the
  measured acceptance reference, recording heard onset versus reported start
  for the first two to three words.
- Prior art: the shared paging tests already lock page switching, gap
  holding, and trailing hold; the transcriber tests already lock token
  grouping, clamping, and the missing-timestamp error.

## Out of Scope

- VAD-gated transcription (Silero model, speech-segment pre-filtering, new
  cgo binding).
- Changing max-initial-timestamp or any parameter the Go binding does not
  expose.
- Model upgrades or swaps.
- New caption animations or visual restyling.
- Reworking segment splitting or caption page sizing.

## Further Notes

- Root causes verified in code: the shared page resolver defaults to page
  zero before any word starts and the segment gate uses the segment start, so
  a lead gap of words[0].start minus segment.start renders early; the
  karaoke template has no per-word time filter and the pop empty mount
  remains during the lead. Upstream docs: word-level timestamps are
  experimental and gated on token probability thresholds; the CLI exposes
  this as the word-threshold flag; VAD is the documented path for discarding
  silence before decoding. Vendored copy is upstream commit d1f114da with
  max-initial-timestamp 1.0, token thresholds 0.01, no-speech threshold 0.6,
  VAD off.

## Verification (2026-09-06)

Results recorded on the GitHub issue: the onset gate holds on real whisper
output (locked in CI by the fixture test), but the vendored build stamps the
first tokens inside the fixture's leading silence (reported 0.00 s vs measured
3.10 s), so caption-during-silence remains possible when the transcriber does
that — upstream of the gate, VAD is the documented upgrade path. Fixture:
`server/testdata/onset-fixture/` (regenerable via `server/cmd/onsetfixture`).
