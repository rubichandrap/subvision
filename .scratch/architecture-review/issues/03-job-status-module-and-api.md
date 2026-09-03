# 03 — job-status-module-and-api

**GitHub issue:** #4

**Status:** ready-for-agent

## Parent

#1

## What to build

A job's Process lifecycle (uploaded → transcribing → rendering → done/failed) exists as real server-side state. The server consumes the JobCompleted event and records transitions; the state is exposed through read-only endpoints `GET /jobs` and `GET /jobs/:id`, each job response including its current stage and the download URL for its Output when done. The persistence mechanism is chosen during implementation — revive the dormant database package or keep in-memory state with documented trade-offs — and the choice is recorded as an ADR so future reviews do not re-litigate it.

## Acceptance criteria

- [ ] An end-to-end run's status is queryable and reflects the real lifecycle stages
- [ ] A done job's response includes a working download URL for its Output
- [ ] Unknown job ids return a 404; malformed requests are rejected
- [ ] The persistence choice is recorded as an ADR (or a decision note on this issue)

## Blocked by

- #3 — the completion event that feeds this module is published by the render module.
