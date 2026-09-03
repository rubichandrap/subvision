# 04 — real-process-tracking-client

**GitHub issue:** #5

**Status:** ready-for-agent

## Parent

#1

## What to build

The client's processes list and detail pages read real Process state from the status API instead of localStorage, poll it while a job is in flight, and download via the API-provided URL. All fabricated ids, sample data, and simulated progress are deleted — the client invents nothing the server owns.

## Acceptance criteria

- [ ] Uploading a video shows its real Process moving through the pipeline stages
- [ ] The download button delivers the actual rendered video
- [ ] No fabricated process ids, sample data, or timer-based simulated progress remain
- [ ] The pages handle in-flight, done, and failed states from the API

## Blocked by

- #4 — the status API must exist for the client to read.
