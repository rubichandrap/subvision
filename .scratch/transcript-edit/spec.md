Promoted: #30

# Spec: Transcript view, edit, and re-render

## Problem Statement

Whisper's word timings drift by a fraction of a second: a Timed Word appears
before it is spoken, or arrives late. Decoder-side tuning (beam width,
entropy threshold, larger model — ADR-0008) did not move the needle, and DTW
plus built-in VAD are out of reach without editing vendored code (ADR-0007).
The user has no way to see what the transcriber produced, fix it, and get a
corrected video. Today the only path is re-uploading the source and hoping
for a better decode.

## Solution

After transcription, the user opens the Process detail page, sees every
Transcription Segment with its text and timestamps, edits text and per-segment
timing, and re-renders. The Output is replaced in place; the Process returns
to done when the new render completes.

## User Stories

1. As a video owner, I want to see the transcribed text of my video, so that I know what the captions will say.
2. As a video owner, I want to see each Transcription Segment's start and end time, so that I know when each caption appears.
3. As a video owner, I want to see the transcript only after transcription finished, so that I never edit half-decoded text.
4. As a video owner, I want to edit a segment's text, so that wrong words become right words.
5. As a video owner, I want to shift a segment's start time, so that an early caption appears on time.
6. As a video owner, I want to shift a segment's end time, so that a late caption lands correctly.
7. As a video owner, I want invalid timing rejected (end before start, negative times), so that I cannot break the render.
8. As a video owner, I want to re-render after editing, so that the Output carries my corrections.
9. As a video owner, I want the Process to show rendering while my re-render runs, so that the status never lies about what is happening.
10. As a video owner, I want my original Edit Spec (trim, Frame, Animation, Subtitle Style) applied to the re-render, so that only my text and timing changes take effect.
11. As a video owner, I want the re-rendered video at the same download link, so that nothing else in my workflow changes.
12. As a video owner, I want failed re-renders to surface a reason, so that I know what went wrong.
13. As a video owner, I want to edit again after a re-render, so that correction is iterative until the captions are right.

## Implementation Decisions

- Transcription Segments are persisted per Upload in a new SQLite table in
  the job module (today they are transient, living only inside the VFX Job
  message). The processor writes the whisper-original segments at transcribe
  time; user edits overwrite them.
- A new read endpoint serves the segments for one Process; a new write
  endpoint accepts edited segments (text plus per-segment start/end) with
  validation (finite numbers, start >= 0, end > start, ordered segments).
- A new store method reopens a done Process into rendering, bypassing the
  terminal-stage guard (which stays for the normal pipeline). The Tracker
  interface gains the same method.
- Re-render publishes a fresh VFX Job with the edited segments plus the
  stored Edit Spec from the original upload. The vfx contract is unchanged —
  it cannot tell an original render from a re-render.
- The Output overwrites `outputs/<id>` in place (last-writer-wins, no
  versioning — no version concept exists in the domain and none is created).
- The client gains a Transcript card inside the Process detail view, between
  the status card and the video block: read-only text while transcribing,
  editable once rendering or done. It rides the existing per-Process poll.
- Timed Words inside an edited segment keep their whisper-original relative
  offsets, scaled to the edited segment window; word timings are never
  re-derived from scratch.
- Render of an unedited Process is byte-identical in behavior to today: the
  persisted segments are the whisper originals.

## Testing Decisions

- Test external behavior only: segment persistence round-trip, endpoint
  validation rejects bad timing, reopen transitions done to rendering and
  refuses unknown ids, re-render publishes a job carrying edited segments.
- Test at the store, handler, and processor seams — the same seams the
  existing job, handler, and processor tests cover.
- The client Transcript card follows the existing component test pattern;
  the acceptance fixture stays the decode-accuracy gate, unchanged.

## Out of Scope

- Per-word timing edits (start/end per Timed Word). Segment-level only; per-word comes later if segment shifts prove insufficient.
- Edit Spec editing (trim, Frame, Animation, Subtitle Style) on re-render. The original spec is reused as-is.
- Output versioning or history. Overwrite in place; no new domain concept.
- Token-probability logging (ADR-0008 step 4). Dropped: probabilities measure word correctness, not timing drift.
- Export to JSON/SRT files. The in-app editor replaces the need.

## Further Notes

- Grill decisions Q1–Q8 (2026-09-07): segment-level edit, original Edit Spec
  reused, overwrite in place, one knob at a time with revert rule.
- ADR-0008 records the failed decoder-tuning measurements that motivate this
  spec: beam 8, entropy 2.0, and small.en each tested with no significant
  change on the reporter's video.
