Promoted: #45

# Spec: Collapse Process lifecycle and transcript orchestration into a deep Job module

## Problem Statement

The Process lifecycle and transcript orchestration are split across the codebase. While ADR-0002 established that `internal/job` owns the Process lifecycle, the Job module was left as a shallow SQL repository exposing 13 individual database operations. The HTTP transport handler was forced to absorb domain orchestration: validating segment timing invariants, rescaling word offsets against stored originals, managing stage checks for editing and re-rendering, publishing re-render jobs to RabbitMQ with rollback on failure, and coordinating database row deletion with storage object cleanup.

This leaks domain rules into HTTP routing, requires the HTTP handler to take six separate interfaces, causes `main.go` to inject the same dependencies multiple times through artificial interfaces, and forces domain verification tests to spin up an HTTP router, format fake requests, and parse JSON responses.

## Solution

Deepen the Job module so it directly owns the Process lifecycle, transcript edits, re-render coordination, and deletion cleanup behind a cohesive interface. The HTTP transport handler becomes a thin adapter that only parses HTTP requests, calls the Job module, and writes HTTP responses. Pure mathematical transformations on Transcription Segments (timing validation, word rescaling) move to the transcriber module alongside the domain types. Side-effect ports for publishing render jobs and cleaning storage objects are injected into the Job module, allowing domain tests to cross the Job interface directly using an in-memory SQLite store and fake publisher without HTTP routing ceremony.

## User Stories

1. As a video owner, I want my edited transcript segments validated for valid positive times and chronological order, so that malformed timing cannot corrupt subtitle rendering.
2. As a video owner, I want words inside my edited segments to maintain their relative spoken timing, so that moving a segment window doesn't destroy karaoke word synchronization.
3. As a video owner, I want to re-render my video after editing transcript segments, so that my corrections are burned into a new output video.
4. As a video owner, I want re-rendering to be rejected if my job is not in the done stage, so that I cannot re-render an in-flight or failed process.
5. As a video owner, I want a failed re-render publish to mark my process as failed with a clear reason, so that I am never left waiting on a phantom render.
6. As a video owner, I want deleting a process to remove its record immediately, so that the gallery updates without waiting on storage operations.
7. As a video owner, I want deleting a process to clean up uploaded and rendered video files best-effort, so that deleted videos do not accumulate storage waste.
8. As a video owner, I want storage cleanup failures during deletion logged but ignored, so that a transient storage glitch never prevents me from deleting an unwanted process.
9. As a maintainer, I want the Job module to own all Process state transitions, so that the status state machine has exactly one source of truth.
10. As a maintainer, I want transcript validation and word offset rescaling to live as pure functions in the transcriber module, so that mathematical timing transformations are isolated from database and network I/O.
11. As a maintainer, I want the HTTP transport handler to act as a thin adapter, so that transport concerns (routing, status codes, JSON serialization) never mix with domain logic.
12. As a maintainer, I want domain failures exposed as typed sentinel errors, so that the HTTP adapter maps them to HTTP status codes without the domain module knowing about HTTP.
13. As a maintainer, I want six shallow handler interfaces collapsed into two clean interfaces, so that wiring in main.go does not pass the same instances repeatedly through fragmented interfaces.
14. As a maintainer, I want Process deletion to delete database records first before cleaning storage, so that UI state is immediately consistent per ADR-0004.
15. As a test author, I want to verify Process lifecycle, transcript saving, and re-rendering directly against the Job module interface, so that tests do not require spinning up an HTTP server or parsing HTTP responses.
16. As a test author, I want to test segment timing validation and word rescaling with pure unit tests, so that timing math tests execute instantly without database setup.
17. As a test author, I want the Job module testable against an in-memory SQLite database, so that tests remain fast, hermetic, and independent of disk state.
18. As the client application, I want transcript save requests to return 400 with validation details when timings are invalid, so that editing errors can be shown to the user.
19. As the client application, I want transcript save requests to return 409 when the process is in a non-editable stage, so that late edits on in-flight processes are refused cleanly.
20. As the client application, I want re-render requests to return 409 when the process is not done, so that invalid re-render triggers fail predictably.

## Implementation Decisions

- The Job module directly encapsulates SQLite access, the Process lifecycle state machine, transcript persistence, re-render coordination, and deletion cleanup.
- External side-effect dependencies are defined as injected ports on the Job module: a publisher port for enqueueing VFX Jobs and an object cleaner port for erasing storage prefixes.
- Pure functions for segment timing validation (non-finite values, negative starts, non-positive durations, and overlapping/out-of-order bounds) and proportional word timestamp rescaling live in the transcriber module alongside the domain types.
- The HTTP handler exposes only routing and translation: it consumes a single Process manager interface and an output streaming interface. The six fragmented interfaces are removed.
- Domain errors are returned as typed sentinel and value errors (not-found, stage-conflict, validation-failure); the HTTP handler translates these into corresponding HTTP status codes (404, 409, 400).
- Re-render coordination: loads the stored segments and original Edit Spec, reopens the Process to rendering, publishes the VFX Job, and rolls back the Process to failed if publishing fails.
- Deletion coordination: removes the database records first (Process, segments, edit spec), then invokes the storage cleaner best-effort for upload and output prefixes, logging any storage errors without failing the operation, adhering strictly to ADR-0004.
- All database schema tables and queries remain private implementation details of the Job module.

## Testing Decisions

- Good tests check external behavior through the module interface, never internal implementation details (such as specific SQL queries or unexported helper functions).
- Primary test surface: the Job module interface, using an in-memory SQLite database (`:memory:`) and an in-memory fake publisher.
- Pure functions in the transcriber module (`ValidateSegmentTiming`, `RescaleWords`) are tested via deterministic unit tests across edge cases (zero duration, negative offsets, proportional scaling).
- HTTP handler tests are reduced to lightweight route-mapping smoke tests checking that HTTP status codes and payloads map correctly to/from the domain interface.
- Existing tests in the HTTP test file that tested domain validation, word rescaling, and re-render rollback through HTTP requests are replaced by direct domain tests on the Job module.
- Prior art: existing in-memory SQLite tests in `server/internal/job/job_reopen_test.go` and `job_segments_test.go`.

## Out of Scope

- Modifying the VFX service or the Remotion rendering templates.
- Changing the wire contract for VFX Jobs on RabbitMQ.
- Client-side changes to the Transcript card or Process gallery.
- Soft deletes or process archiving (deletion remains immediate and irreversible per ADR-0004).
- Editing the original Edit Spec during re-rendering (the stored spec is reused as-is).

## Further Notes

- Originates from Candidate 1 of the 2026-09-07 architecture review.
- Aligns with and reinforces ADR-0002 (SQLite for job state) and ADR-0004 (Immediate best-effort delete).
- `CONTEXT.md` updated to reflect that a Process allows transcript editing and re-rendering.
