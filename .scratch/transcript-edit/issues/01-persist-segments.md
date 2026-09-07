Promoted: #32

# 01 — Persist Transcription Segments per Upload

**What to build:** After whisper transcribes an upload, the original segments are stored in the database owned by the job module, keyed by the upload id, so they outlive the queue message and can be re-read later. Deleting a process removes its segments too. Nothing about the render path changes: the published job carries the same segments as today.

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

- [ ] Whisper-original segments stored per upload at transcribe time
- [ ] Stored segments readable after the job reaches done (store-level round trip)
- [ ] Delete removes a process's segments along with its record
- [ ] Unedited renders behave exactly as today
