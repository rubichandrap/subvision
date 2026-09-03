# 01 — reconnect-vfx-job-handoff

**GitHub issue:** #2

**Status:** ready-for-agent

## Parent

#1

## What to build

The server and the vfx service agree on one VFX Job contract: the queue name and the payload shape are each defined in exactly one place per runtime, and the publisher's name matches the subscriber's. A VFX Job published after transcription is actually received by the vfx service. A malformed or unexpected payload fails loudly — logged with enough context to debug, never silently dropped.

## Acceptance criteria

- [ ] Publishing a VFX Job from the server delivers it to the vfx service (visible in the queue's management UI)
- [ ] Queue name and payload shape are each defined once per runtime and derived from that definition
- [ ] A malformed payload produces a clear error log and is not silently dropped
- [ ] The transcription's Transcription Segments reach the vfx service unchanged

## Blocked by

None — can start immediately.
