# ADR-0005: Long segments split at transcribe time; word animations page at render time

Date: 2026-09-04 · Status: accepted

## Context

Whisper returns long Transcription Segments (15–20 words are common). The
pop and karaoke animations render every Timed Word of the active segment at
once, so a long segment becomes a wall of text that fills the frame and
overlaps itself. Fade and slide render the full segment text too, so they
suffer the same way.

## Decision

- The transcriber splits long segments using whisper's word gaps (a pause
  starts a new segment) bounded by a max word count and a max duration.
  Split segments keep their original whisper timestamps — nothing is
  re-timed or guessed. This closes README TODO 2.
- The pop and karaoke templates group the active segment's Timed Words into
  Caption Pages of N words (user-configurable through the Edit Spec,
  default 4) and show only the page holding the currently spoken word.

## Alternatives considered

- **Split only, no paging** — fixes fade/slide, but a segment just under
  the split limit is still an 8-word block on a narrow 9:16 frame.
  Rejected as the whole fix; kept as half of it.
- **Paging only, no split** — fixes pop/karaoke display, but fade/slide
  still show full long segments and the stored segments stay unwieldy.
  Rejected as the whole fix; kept as half of it.

## Consequences

- Split and paging are two layers fixing two things (data shape vs screen
  layout), not two fixes for one thing; either alone leaves a wall of text
  somewhere.
- The Edit Spec gains an optional `captions.wordsPerPage` (default 4, range
  2–8). Jobs published without it render with the default, so old payloads
  keep working.
- New domain term: Caption Page (see CONTEXT.md).
