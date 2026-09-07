Promoted: #53

## Parent

Part of #50

## What to build

Migrate core domain consumers (`internal/job`, `internal/vfxjob`, and `internal/handler`) to import `internal/transcript` instead of `internal/transcriber`. `job.Store` validates and rescales edited segments via `transcript.ValidateTiming` and `transcript.RescaleWords`. `vfxjob.Job` serializes `transcript.Segment` slices. The HTTP jobs handler maps `*transcript.ValidationError` to HTTP 400. Database and HTTP packages are completely decoupled from whisper CGO dependencies.

## Acceptance criteria

- [ ] `internal/job/job.go` imports `internal/transcript` and no longer imports `internal/transcriber`
- [ ] `internal/vfxjob/vfxjob.go` imports `internal/transcript` and no longer imports `internal/transcriber`
- [ ] `internal/handler/jobs.go` imports `internal/transcript` and no longer imports `internal/transcriber`
- [ ] `job.Store.SaveSegments` uses `transcript.ValidateTiming` and `transcript.RescaleWords`
- [ ] `handler.respondWithError` maps `*transcript.ValidationError` to HTTP 400 with failure details
- [ ] Domain tests in `job`, `vfxjob`, and `handler` pass without whisper CGO flags

## Blocked by

- #51 — Pure domain transcript module and timing invariants
- #52 — Deep transcriber audio adapter
