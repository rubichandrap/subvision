Promoted: #58

# 02 — Atomic Ingestion Commit and Unified VFX Job Publishing

**What to build:** An atomic macro-transition on the Job store that saves Transcription Segments, saves the Edit Spec, transitions the Process to the rendering stage, and publishes the VFX Job in a single guarded operation with automatic rollback to failed if queue publishing fails.

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

- [ ] `job.Store.CommitIngestion(ctx, id, segments, spec)` persists Transcription Segments to `job_segments`.
- [ ] `CommitIngestion` persists the Edit Spec to `job_edit_specs` (clearing any old copy if nil).
- [ ] `CommitIngestion` updates `jobs.stage` to `rendering` within the same transaction, guarded against already-terminal jobs.
- [ ] `CommitIngestion` dispatches the `vfxjob.Job` to RabbitMQ via the store's injected publisher.
- [ ] If publishing fails, the Process rolls to `failed` with the error reason recorded, matching re-render error handling.
- [ ] Calling `CommitIngestion` on an unknown or terminal job returns a conflict/error without corrupting state.
- [ ] In-memory SQLite unit tests verify atomic success, conflict guards, and publish failure rollback.
