Promoted: #57

# 01 — Pure Transcript Timestamp Shifting

**What to build:** A pure domain function in the Transcript module that shifts Transcription Segment boundaries and all contained Timed Word timestamps by an offset in seconds, translating trim-window-relative timestamps into absolute source video coordinates without mutating input structures or touching I/O.

**Blocked by:** None — can start immediately.

**Status:** done

- [x] `transcript.Shift(segments []Segment, offsetSeconds float64) []Segment` translates start and end times of every segment by `offsetSeconds`.
- [x] Every Timed Word contained in each segment has its start and end times translated by `offsetSeconds`.
- [x] Word-level relative intervals and ordering are strictly preserved.
- [x] Empty segment slices and segments without words pass through cleanly without panic.
- [x] Zero offset returns equivalent segments and words.
- [x] Unit tests exercise zero, positive, and fractional second offsets with no disk or database dependencies.
