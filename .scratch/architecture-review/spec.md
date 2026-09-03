# Spec — make the subtitle pipeline real

**GitHub issue:** #1 (label: ready-for-agent)

**Tickets:** #2 → #3 → #4 → #5 · #6 · #7 (blocking edges set natively on GitHub)

---

## Parent

Architecture review, 2026-09-03 (implemented as a series of child issues). Storage migration decision: ADR-0001.

## Problem Statement

A creator uploads a video and expects subtitles, but everything after the upload is unreliable or fake: the job that should trigger subtitle rendering is silently never received by the rendering service, the render path itself cannot complete, the "track your process" pages simulate progress with random numbers, and the download button delivers nothing. Meanwhile the three services each invent their own configuration vocabulary, and the codebase carries a second, abandoned architecture that every reader must pay to understand.

## Solution

Make the pipeline real end to end. Reconnect the job handoff with a contract defined once per runtime, concentrate all render-path complexity into one deep render module, give a job's lifecycle real server-side state exposed by a small status API, let the client track that real state and download real outputs, collapse configuration into one documented contract, and delete the ghost pipeline. Tests cross exactly three seams: the job queue (write side), the status API (read side), and the storage client (in-process, faked in tests).

## User Stories

1. As a creator, I want my uploaded video to actually flow through subtitle generation, so that I receive real subtitles instead of a stalled process.
2. As a creator, I want the process page to show my job's real current stage (transcribing, rendering, done), so that I know whether to wait or come back later.
3. As a creator, I want to download the rendered video with burned-in subtitles once the job completes, so that the product delivers its core promise.
4. As a creator, I want a failed job to surface as a clear failure state, so that I am not left watching a fake progress bar.
5. As a maintainer, I want the VFX Job contract defined in exactly one place per runtime, so that a rename cannot silently sever the pipeline again.
6. As a maintainer, I want malformed job payloads to fail loudly, so that contract drift is visible instead of swallowed.
7. As a maintainer, I want failed render jobs to be requeued, so that one bad video cannot wedge the worker.
8. As a maintainer, I want repeatedly failing jobs to dead-letter after a bounded number of attempts, so that the queue self-heals.
9. As a maintainer, I want temporary paths, output naming, and render options owned by the render module, so that caller-side path bugs cannot recur.
10. As a maintainer, I want a job's lifecycle kept as real server-side state, so that the client never invents state.
11. As a maintainer, I want the persistence choice for job state recorded as a decision, so that future reviews do not re-litigate it.
12. As the client app, I want one read-only status interface, so that tracking is a thin polling surface with a single source of truth.
13. As a maintainer, I want the simulated tracking data removed, so that there is exactly one representation of a Process.
14. As an operator, I want one documented environment contract, so that booting the stack is copy-paste from a single page.
15. As a maintainer, I want configuration loaders that fail loudly at boot, so that misconfiguration never hides behind silent defaults in production.
16. As an agent, I want the abandoned synchronous pipeline deleted, so that navigating the codebase means reading one architecture, not two.
17. As a maintainer, I want tests only at the three confirmed seams, so that refactors stay cheap and tests never pin implementation details.

## Implementation Decisions

- The **job contract** — queue name and payload shape — is defined once per runtime. The payload carries the upload id, the upload's object key, and its Transcription Segments; the animation-type field is reserved but unused. Consumers validate payloads; a parse failure logs a clear error and is not silently dropped.
- The **render module** in the vfx service owns id derivation from the object key, all temporary paths (including file extensions), render options (frame rate, dimensions), the ffmpeg invocation, and the Output upload. Callers pass a job and receive an outcome; they make no path or option decisions.
- **Completion semantics**: a successful render publishes a JobCompleted event (upload id + output key); a failed job is requeued; after a bounded number of attempts it dead-letters. Consumer prefetch is configuration, not a machine-specific constant.
- The **Job module** in the server owns the Process lifecycle (uploaded → transcribing → rendering → done/failed), consumes JobCompleted, and exposes read-only status over `GET /jobs` and `GET /jobs/:id`, including the download URL for the Output. The persistence mechanism is chosen during implementation and recorded as a decision (revive the dormant database package vs. in-memory state with documented trade-offs).
- The **client** reads Process state only through the status interface and downloads only via the API-provided URL. All fabricated ids, sample data, and simulated progress are deleted.
- The **environment contract** (`S3_*`, `RABBITMQ_*`, `PORT`, `TMP_DIR`, `CLIENT_URL`, `WHISPER_MODEL_PATH`) is documented in one place; each runtime keeps a thin, typed loader that fails loudly on missing required variables.
- The **ghost pipeline** is deleted: unused server packages, the commented-out synchronous subtitle path, an unused handler parameter, and duplicated client hooks. The SRT writer goes with it — git preserves it if a plain-SRT output ever returns.
- Per ADR-0001, all storage access stays behind the S3 seam; no vendor name appears in any module interface.

## Testing Decisions

- A good test checks external behavior only — never internals. Tests cross exactly three seams: the VFX-job queue, the status API, and the storage client.
- The render path is verified end to end at the queue seam: a synthetic VFX Job results in an Output object at `outputs/<id>`; a malformed payload produces a visible failure, not silence.
- Processor-level tests run against a fake storage client, so no live object store is needed for unit work.
- The status API is tested over HTTP: lifecycle events move a job through its states; unknown ids 404.
- Prior art: none — the repo has no tests today; these establish the pattern (the Go standard testing package on the server; the vfx service's standard runtime test runner).
- The client is verified against the API seam plus a manual UI pass; client test infrastructure is out of scope.

## Out of Scope

- User subtitle preferences (language, color, outline, border, font, size, vertical margin) — a separate feature needing its own spec.
- Splitting Transcription Segments by max characters or seconds; user-configurable video resizing.
- Transcription performance (the whisper model is reloaded per call — flagged for a future ticket).
- Authentication, multi-tenancy, RustFS multi-node or TLS setups, and CI pipeline setup.

## Further Notes

- Origin: the 2026-09-03 architecture review. The storage seam work (review candidate 1) is already delivered and recorded in ADR-0001; the remaining six candidates are ticketed as child issues with blocking edges, in dependency order.
- The review's report file is ephemeral (`/tmp`); this issue plus the child issues are the durable record.
- Fixing the handoff activates the render path for the first time — the render module ticket must land with or before the reconnection goes live, per the review's warning.
