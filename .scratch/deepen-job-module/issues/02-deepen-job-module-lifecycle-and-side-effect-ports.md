Promoted: #47

## Parent

Part of #45

## What to build

The Job module directly owns the Process lifecycle state machine, transcript persistence, re-render execution, and deletion cleanup behind a cohesive interface with internal SQLite and injected ports. Saving segments validates timing and rescales words before persisting to the database. Re-rendering a completed process transitions it back to rendering, publishes a fresh VFX Job to RabbitMQ with the stored Edit Spec, and rolls back to failed if publishing fails. Process deletion immediately removes all database records and triggers best-effort storage cleanup for upload and output objects per ADR-0004.

## Acceptance criteria

- [ ] `job.Store` accepts injected `VfxPublisher` and `ObjectCleaner` ports at construction
- [ ] `job.Store.SaveSegments` enforces that the Process is in `rendering` or `done` stage, validates timings, rescales words against whisper originals, and saves to `job_segments`
- [ ] `job.Store.SaveSegments` returns a typed stage conflict error if the process is in an uneditable stage
- [ ] `job.Store.Rerender` verifies the Process is in `done` stage, loads stored segments and Edit Spec, transitions stage to `rendering`, and publishes a fresh `vfxjob.Job`
- [ ] If re-render publish fails, `job.Store.Rerender` transitions the process to `failed` with the error reason and returns an error
- [ ] `job.Store.Delete` deletes database rows first (`jobs`, `job_segments`, `job_edit_specs`), then invokes `ObjectCleaner` for upload and output prefixes best-effort per ADR-0004
- [ ] Storage cleanup errors during deletion are logged but do not fail the `Delete` operation
- [ ] Domain tests verify save, re-render, rollback, and deletion directly across the `job.Store` interface using in-memory SQLite and fake ports

## Blocked by

- #46 — Pure segment timing validation and word rescaling in transcriber
