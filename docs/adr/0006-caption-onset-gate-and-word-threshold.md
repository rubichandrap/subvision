# ADR-0006: Gate captions on first-word onset; tune word timestamp threshold via existing binding

Date: 2026-09-05 · Status: accepted (no-VAD stance superseded by ADR-0007)

Caption pages in the karaoke and pop animations could appear during leading silence because rendering was gated on the segment start while the first word started later. We now gate every word-driven caption on the first word's start time, and we tune the word timestamp probability threshold through the setters the Go binding already exposes. No model change, no VAD: the vendored Go binding exposes zero VAD API, so VAD would mean a new cgo binding plus a model download, which is disproportionate to this fix. `max_initial_ts` stays at its upstream default for the same reason — the binding does not expose it.

## Consequences

- A caption page never mounts before its first word starts, even when the segment window opens earlier. Wordless segments render unchanged.
- Fade and slide render the segment's text rather than words, but sit behind the same gate: their entrance eases in from the onset, so a late first word never spends the entrance invisibly during leading silence.
- The timestamp threshold becomes a named constant defaulting to the upstream default; it moves only on measured evidence from the fixture video.
- VAD remains the documented upgrade path if threshold tuning proves insufficient (see Further Notes in the spec).
