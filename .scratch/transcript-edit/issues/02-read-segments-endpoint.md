Promoted: #33

# 02 — Read segments endpoint

**What to build:** The status API serves the stored segments for one process, so any client can fetch the transcript without touching the queue or the renderer.

**Blocked by:** 01 — Persist Transcription Segments per Upload.

**Status:** ready-for-agent

- [ ] Endpoint returns the stored segments for a rendering or done process
- [ ] Unknown id returns not-found, consistent with the existing detail endpoint
- [ ] A process with nothing stored yet returns an empty list, not an error
- [ ] Segment shape matches what the render job carries (segments with timed words)
