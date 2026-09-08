Promoted: #56

# Spec: Collapse Ingestion Pipeline and Encapsulate Process Lifecycle Transitions

## Problem Statement

The ingestion pipeline—the server workflow that takes an Upload, extracts its audio, transcribes speech, records the Transcript in the database, and dispatches a VFX Job—suffers from severe architectural friction:

1. **Leaking Lifecycle Seam**: The Job module defines a fine-grained, procedural tracker interface exposing five micro-methods (`MarkTranscribing`, `SaveOriginalSegments`, `SaveEditSpec`, `MarkRendering`, `MarkFailed`). Instead of presenting an atomic operation, the store requires callers to coordinate multi-step persistence sequences manually.
2. **Shallow Processor Coordinator**: The processor module is a thin pass-through that does not own its domain invariants or its failure modes. It downloads videos, shells out to ffmpeg, invokes transcription, loops over structs to shift timestamps, marshals JSON, calls the five tracker methods step-by-step, and publishes to RabbitMQ.
3. **Main Entrypoint as Failure Coordinator**: Because the processor module does not own its failure transitions, the server entrypoint (`main.go`) is forced to catch errors from payload extraction, Edit Spec parsing, and file processing, manually calling a `failJob` helper to mark the Process failed in SQLite.
4. **Split VFX Job Publishing**: Publishing render jobs to RabbitMQ is fragmented across two separate packages: initial uploads are published by the processor, while re-renders are published by the Job module. If publishing fails during a re-render, the Job module rolls the Process to failed; if it fails during an initial upload, the processor bubbles the error up to `main.go`.
5. **Scattered Domain Math & Resource Leaks**: Timestamp coordinate shifting across trim windows is written as an inline struct-mutating loop inside the processor rather than a pure domain function, and temporary audio and video scratch files are never removed from the filesystem after processing completes.

## Solution

Collapse the ingestion pipeline into an autonomous, deep Ingestion module that owns the entire upload-to-render workflow behind a single, high-leverage interface.

The Job module's interface shrinks by replacing the procedural micro-methods with an atomic commit method (`CommitIngestion`) that executes segment persistence, Edit Spec storage, transition to the rendering stage, and VFX Job queue dispatch inside a single guarded operation. If publishing fails, the Process rolls to the failed stage automatically.

The Ingestion module accepts an upload job payload, parses and validates the Edit Spec, coordinates video download, audio extraction, and transcription via clean adapters, shifts transcript timestamps using a pure domain function in the Transcript module, commits the results to the Job store, and ensures immediate cleanup of working scratch files. An autonomous failure guard inside the pipeline ensures any failure (validation, download, extraction, transcription, or commit) immediately transitions the Process to failed with the exact cause, removing error coordination from the server entrypoint.

## User Stories

1. As a video creator uploading a video, I want my upload processed through speech transcription and queued for rendering automatically, so that subtitles are generated without manual intervention.
2. As a video creator uploading a video with a trim window, I want transcription timestamps shifted into absolute video coordinates, so that captions remain synchronized with my trimmed clip.
3. As a video creator with an invalid Edit Spec or corrupted video, I want my process marked as failed immediately with a clear reason, so that I am never left waiting on an indefinitely stalled render.
4. As a video creator whose upload encounters a server-side failure during download, extraction, or transcription, I want the failure reason recorded on my Process, so that I can see why my job did not succeed.
5. As an API client polling a Process, I want stage transitions (`uploaded` → `transcribing` → `rendering` → `done`/`failed`) to be strictly monotonic and guarded against invalid overwrites, so that status progress reflects real backend state.
6. As a maintainer, I want the Job store to provide an atomic `CommitIngestion` operation, so that a Process cannot enter the rendering stage without its segments and Edit Spec already saved and its VFX Job dispatched.
7. As a maintainer, I want VFX Job publishing unified under the Job module for both initial uploads and re-renders, so that render queue dispatch logic is never duplicated.
8. As a maintainer, I want the Ingestion pipeline to own Edit Spec parsing and parameter validation, so that the server entrypoint does not act as a domain validator.
9. As a maintainer, I want failure state transitions to be self-contained within the Ingestion pipeline, so that the server entrypoint has zero responsibility for error recovery.
10. As a maintainer, I want the shallow processor module deleted, so that unnecessary abstraction layers and intermediate pass-throughs are eliminated.
11. As a maintainer, I want five procedural lifecycle micro-methods deleted from the Job store's external interface, so that internal database update steps stop leaking across module seams.
12. As a maintainer, I want transcript timestamp shifting to live as a pure domain function in the Transcript module, so that coordinate math is tested independently of file and network I/O.
13. As a maintainer, I want temporary video and audio scratch files deleted immediately after ingestion completes or fails, so that long-running server instances do not exhaust local disk storage.
14. As an operator deploying Subvision, I want temporary scratch files to have strict lifecycle boundaries, so that container storage limits are not exceeded over time.
15. As a test author, I want to verify the entire Ingestion pipeline through a single method call using in-memory adapters, so that integration tests run in milliseconds without real ffmpeg or whisper binaries.
16. As a test author, I want to test timestamp coordinate shifting with deterministic unit tests, so that edge cases like zero duration, fractional seconds, and word boundary preservation are thoroughly exercised.
17. As a test author, I want to test atomic ingestion commit and publish rollback against an in-memory SQLite store, so that database transition guarantees are verified hermetically.
18. As a test author, I want pipeline failure guard behavior tested across all failure points (storage, extraction, transcription, publishing), so that no error path leaves an in-flight process unhandled.
19. As a developer modifying the server entrypoint, I want consumer callbacks to be a one-line forward to the Ingestion pipeline, so that wiring remains clean and decluttered.
20. As a developer navigating the codebase, I want module and interface names to strictly reflect the domain concepts in `CONTEXT.md`, so that system boundaries are immediately clear.

## Implementation Decisions

- **Ingestion Module**: A new deep module (`internal/ingest`) replaces the shallow `processor` package. It presents a single entrypoint:
  `ProcessUpload(ctx context.Context, job UploadJob) error`
- **Upload Job Contract**: The pipeline accepts a structured upload job carrying the upload identifier, object key, and raw Edit Spec metadata. The pipeline directly encapsulates Edit Spec parsing and validation.
- **Autonomous Failure Guard**: Any error occurring during ingestion (missing object key, invalid Edit Spec, download failure, audio extraction failure, transcription failure, or commit error) automatically records the failure on the Process via the lifecycle adapter before bubbling the error up. The `failJob` helper in `main.go` is deleted.
- **Atomic Job Commit**: The Job store (`internal/job`) introduces `CommitIngestion(ctx, id, segments, spec)`. Within a single atomic database transaction, it saves the Transcription Segments to `job_segments`, saves the Edit Spec to `job_edit_specs`, updates `jobs.stage` to `rendering` (guarded against terminal states), and publishes the `vfxjob.Job` to RabbitMQ. If publishing fails, it rolls the Process to `failed` and logs the error, mirroring the re-render path.
- **Job Store Interface Shrinkage**: The micro-methods `SaveOriginalSegments`, `SaveEditSpec`, and `MarkRendering` are removed from the public interface.
- **Pure Coordinate Shifting**: The Transcript module (`internal/transcript`) gains a pure function `Shift(segments []Segment, offsetSeconds float64) []Segment` that returns a fresh slice of segments with start, end, and all contained word bounds translated by the offset. Inline loops in pipeline code are eliminated.
- **Ports & Adapters Architecture**: The Ingestion pipeline accepts four minimal interfaces:
  1. `VideoDownloader`: downloads the video from object storage to a local path.
  2. `AudioExtractor`: extracts 16kHz mono WAV from the video within a trim window.
  3. `AudioTranscriber`: transcribes the WAV file into pure domain Transcription Segments.
  4. `ProcessLifecycle`: manages stage transitions (`StartTranscription`, `CommitIngestion`, `MarkFailed`).
- **Scratch File Locality & Cleanup**: Working video and WAV files are cleaned up immediately via deferred removal once ingestion finishes, regardless of success or failure.
- **Clean Cutover**: The `processor` package is completely removed. Server wiring in `main.go` instantiates the Ingestion pipeline and forwards upload events directly.

## Testing Decisions

- A good test exercises observable behavior through the module's public interface, verifying state changes and published events rather than asserting internal helper calls.
- **Primary Seam (Ingestion Pipeline Interface)**: Tests invoke `ingest.Pipeline.ProcessUpload` with an `UploadJob`. Adapters are satisfied with in-memory fakes:
  - Success cases verify that the Process reaches `rendering`, segments and Edit Spec are committed, and the VFX Job is published with window-shifted timestamps.
  - Failure cases (invalid key, corrupt Edit Spec, download error, extraction error, transcription error, commit error) verify that the Process transitions to `failed` with the expected reason.
  - Scratch file cleanup is verified by ensuring temporary paths do not exist after the call returns.
- **Secondary Seam (Pure Transcript Domain)**: Unit tests for `transcript.Shift` verify translation across positive, zero, and fractional offsets for segments with words, segments without words, and empty slices without disk or database access.
- **Tertiary Seam (Job Store Commit)**: Tests for `job.Store.CommitIngestion` using an in-memory SQLite store (`:memory:`) verify atomic state updates, conflict guards on terminal processes, and rollback to `failed` when the publisher adapter fails.
- **Prior Art**: In-memory SQLite tests in `server/internal/job/store_test.go` and `job_reopen_test.go`; contract tests in `server/internal/vfxjob/vfxjob_test.go`.

## Out of Scope

- Modifying the VFX service (`vfx/`) or Remotion subtitle templates.
- Changing the RabbitMQ wire format for VFX Jobs.
- Altering tus upload handling or client-side tus-js-client configuration.
- Client-side changes to the Transcript Card or Process details page.
- Adding distributed job cancellation (deferred per ADR-0004).

## Further Notes

- Originates from Candidate 1 of the 2026-09-07 architecture review.
- Aligns with and reinforces ADR-0002 (SQLite for job state), ADR-0003 (Edit Spec via tus metadata), and ADR-0005 (Segment split and caption paging).
- `CONTEXT.md` was updated inline during the review to record the canonical definition of **Ingestion**.
