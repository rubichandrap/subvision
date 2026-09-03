# ADR-0004: Process deletion is immediate and best-effort; it does not cancel work

Date: 2026-09-03 · Status: accepted

## Context

The gallery is read-only today; users cannot remove old videos. Deleting a
Process means erasing three things: the SQLite row, the Upload object, and
the Output object. But the pipeline is asynchronous — an upload job may be
sitting in RabbitMQ, whisper may be transcribing, or the vfx service may be
rendering while the user presses delete. Coordinating a true cancellation
across all of that is a contract change (ADR-grade) and is deferred.

## Decision

- `DELETE /jobs/:id` hard-deletes the Process row first (source of truth for
  the UI), then removes `uploads/<id>` and `outputs/<id>` best-effort.
- Deletion is allowed at any stage, including in-flight. A racing worker's
  eventual write is discarded: the store rejects transitions for unknown or
  terminal jobs, so the row cannot resurrect.
- No soft delete, no archive. Deletion is irreversible by design.
- Real cancellation (stopping whisper/ffmpeg/render, aborting tus uploads)
  stays a separate feature with its own design.

## Alternatives considered

- **Refuse to delete in-flight jobs (409)** — simplest, but the delete button
  would be dead exactly when users want it (a stuck or unwanted render).
  Rejected.
- **Soft delete (`deleted_at` column)** — buys restore/undo nobody asked for;
  every list and query now filters forever. Rejected.

## Consequences

- A worker that finishes just after object deletion can orphan a fresh
  `outputs/<id>` (the row is already gone, so nothing points at it). Storage
  junk, never a UI ghost; clean manually if it appears.
- Failure to remove an object is logged loudly, not retried — the UI outcome
  (row deleted, list refreshed) is not held hostage to storage errors.
