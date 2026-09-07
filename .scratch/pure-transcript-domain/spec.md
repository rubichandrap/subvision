Promoted: #50

# Spec: Extract Pure Domain Transcript Module from CGO-Coupled Transcriber

## Problem Statement

Core domain entities (Transcription Segment, Timed Word, Transcript) and their pure invariant operations (validating positive chronological segment timings, rescaling word timestamps within edited windows, and splitting speech pause boundaries) are trapped inside the audio transcription module. Because that module wraps the C++ whisper.cpp engine using CGO, every package that touches transcripts—including SQLite database persistence, RabbitMQ wire contracts, and HTTP status routing—is forced to import the audio transcription package and link against CGO dependencies and C++ libraries.

This causes architectural friction across multiple dimensions:
1. Pure domain transformations on subtitles cannot be compiled or tested without configuring CGO build paths and whisper library links.
2. Changes to domain structures trigger re-compilation of heavy audio decoding code.
3. Database persistence (the Job module), messaging payloads (the VFX job module), and HTTP handlers (the jobs status handler) have their boundaries violated by depending directly on an audio decoding package instead of a pure domain model.
4. Future audio adapter swaps or alternative transcription providers are blocked because callers depend on concrete whisper-coupled types rather than an in-process transcription seam.

## Solution

Extract a pure domain module for Transcripts that owns Transcription Segment and Timed Word structures, chronological timing validation, relative word offset rescaling, and speech pause segmentation. This domain module has zero external dependencies and requires zero CGO.

The audio transcription engine becomes a pure adapter behind a clean in-process seam, receiving an audio file and returning pure domain segments. Upstream consumers (database persistence in the Job module, wire contracts in the VFX job module, and HTTP routing) import the pure domain module directly, completely severing their dependency on CGO and whisper bindings. All existing pure segment tests migrate to the domain module and execute instantly in standard Go.

## User Stories

1. As a developer modifying the Job module database tables, I want to import pure transcript data structures, so that my database package never pulls in C++ audio libraries or CGO flags.
2. As a developer working on HTTP status endpoints, I want domain validation errors to come from a pure domain package, so that HTTP transport tests do not require CGO linking.
3. As a developer writing wire contracts for the VFX service, I want Transcription Segments to be defined in a pure domain package, so that contract serialization does not depend on an audio engine.
4. As a test author running unit tests across the repository, I want domain timing validation tests to run with standard Go test commands without setting whisper CGO link environment variables, so that test execution is fast and frictionless.
5. As a test author, I want segment pause-splitting rules tested in isolation from audio decoding, so that speech pause boundary logic is verified deterministically with simple fixtures.
6. As a test author, I want word timestamp rescaling tested through a pure domain interface, so that relative word timing math can be exercised across edge cases without invoking audio models.
7. As a maintainer, I want the audio transcription module to act as an adapter rather than a domain container, so that replacing or upgrading the underlying whisper model never impacts database schemas or HTTP routing.
8. As a maintainer, I want a clean in-process seam between pipeline orchestration and audio transcription, so that mock audio decoders can be plugged in during integration tests.
9. As an operator building container images, I want packages that don't transcribe audio to compile without CGO headers, so that build layers remain lean and decoupled.
10. As a maintainer, I want segment validation to reject non-finite timestamps, negative start times, non-positive durations, and overlapping or out-of-order segment bounds with descriptive error details, so that malformed transcripts are rejected before reaching storage or rendering.
11. As a maintainer, I want word timestamps within edited segments to be proportionally rescaled against whisper originals, so that subtitle animations maintain accurate synchronization when segment windows are adjusted.
12. As a maintainer, I want long transcription segments to be automatically split on natural speech pauses exceeding the pause threshold and capped by word and duration limits per ADR-0005, so that on-screen subtitles remain readable.
13. As an API client, I want segment timing validation errors to clearly identify the failing segment index and reason, so that editing errors can be highlighted to the user.
14. As a video creator editing subtitles, I want edited segments to preserve their word timing relative to speech onset, so that karaoke and pop animations align with the spoken words in the re-rendered video.
15. As a maintainer navigating the codebase, I want domain types to match the vocabulary in CONTEXT.md, so that code structure aligns with project concepts.
16. As a developer running test suites in CI or local environments, I want `go test ./...` in packages like `job`, `vfxjob`, and `handler` to succeed without requiring whisper.cpp pre-compiled artifacts, so that contributor setup is simplified.
17. As an engineer exploring alternative transcription models in the future, I want to implement an adapter that returns domain segments, so that no changes to the Job store or VFX handoff are necessary.
18. As a test author, I want audio decoding tests to focus exclusively on whisper context initialization, token-to-word grouping, and threshold application, so that audio tests don't re-test segment validation math.
19. As a developer maintaining the onset acceptance fixture, I want the CLI tool to use the transcriber adapter cleanly and output pure domain segments, so that acceptance verification remains reliable.
20. As a maintainer, I want zero transitional type aliases or legacy shims left behind in the codebase, so that technical debt is eliminated upon refactoring.

## Implementation Decisions

- A new pure domain package is established at the root of the server internal hierarchy to encapsulate the Transcript domain model and its invariants.
- The Transcript domain package defines the core domain entities:
  - `Segment` representing a Transcription Segment (start time, end time, text, and contained words).
  - `Word` representing a Timed Word (text, start time, end time).
- Invariant domain operations are implemented as pure functions in the Transcript package:
  - Timing validation verifying finite numbers, non-negative start times, positive durations, and strict non-overlapping chronological order, returning a typed validation error.
  - Proportional word timestamp rescaling keeping relative offsets intact within modified segment windows.
  - Speech pause segmentation cutting segments on natural pauses (≥ 0.4 seconds) with maximum word (12) and duration (8.0 seconds) bounds adhering to ADR-0005.
- The audio transcriber module is refactored into a deep adapter:
  - Exposes an adapter struct with a constructor accepting configuration settings.
  - Implements a clean transcription method accepting an audio file path and returning a slice of pure domain segments or an error.
  - Encapsulates all whisper CGO bindings, model loading, memory buffers, and token-to-word translation internally.
  - Runs speech pause segmentation on decoded segments before returning them to callers.
- Domain consumers across the server are migrated in a clean cutover:
  - The Job module imports the Transcript domain package for segment storage, validation during edits, and word rescaling during re-render prep.
  - The VFX Job module imports the Transcript domain package for payload serialization in the render queue message.
  - The HTTP handler imports the Transcript domain package for error mapping of validation failures to HTTP 400.
  - The processor module accepts a transcription interface or function type returning domain segments, decoupling it from whisper implementation specifics.
  - Main application wiring and fixture utilities instantiate the transcriber adapter and inject its transcription method into the pipeline.
- No backward-compatibility aliases, deprecated re-exports, or migration shims are retained.

## Testing Decisions

- A good test checks external observable behavior at the module interface without asserting internal wiring, unexported helpers, or incidental implementation details.
- Primary test surfaces:
  - Pure domain unit tests: verify timing validation boundaries, word rescaling math, and speech pause segmentation logic in the Transcript module with deterministic test cases, running in pure Go without CGO.
  - Audio adapter wiring tests: verify that the transcriber adapter parses WAV headers, sets decoder thresholds, groups whisper tokens into domain words, and handles silent/synthetic audio.
  - Consumer integration tests: existing domain tests in the Job store and VFX contract continue to verify persistence and serialization using domain segments without needing CGO flags.
- Prior art:
  - Existing segment timing tests (`timing_test.go`) and pause split tests (`split_test.go`) in the transcriber package migrate directly to the Transcript package.
  - Existing Job store tests in `store_test.go` and contract tests in `vfxjob_test.go`.

## Out of Scope

- Altering the whisper.cpp submodule, C++ source, or pre-compiled model files.
- Changing the client-side Transcript Card UI or API contract (`GET /jobs/:id/segments`, `PUT /jobs/:id/segments`).
- Changing the JSON wire contract for VFX Jobs sent to the rendering service.
- Modifying decoder threshold parameters or VAD configuration (governed by ADR-0006 and ADR-0007).
- Changing SQLite schema for `job_segments`.

## Further Notes

- Originates from Candidate 2 of the 2026-09-07 architecture review.
- Aligns with ADR-0005 (Caption pages and segment split), ADR-0006 (Caption onset gate and word threshold), and ADR-0007 (VAD and DTW word timestamps).
- `CONTEXT.md` was updated inline during the review to record the canonical definition of **Transcript**.
