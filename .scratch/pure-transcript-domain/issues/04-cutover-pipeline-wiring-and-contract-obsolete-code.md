Promoted: #54

## Parent

Part of #50

## What to build

Migrate pipeline execution (`processor`), entry point wiring (`main.go`), and fixture generation (`onsetfixture`) to the new transcriber adapter seam and domain segments. Remove obsolete domain definitions (`Segment`, `Word`), timing validation, word rescaling, and pause splitting from `internal/transcriber`, ensuring all callers now consume `internal/transcript`. Verify that the entire test suite passes cleanly with zero deprecated shims or leftover dead paths.

## Acceptance criteria

- [x] `processor.TranscribeFunc` signature updated to return `[]transcript.Segment`
- [x] `server/cmd/subvision/main.go` instantiates `transcriber.New(...)` and injects `transcriber.Transcribe` into the processor
- [x] `server/cmd/onsetfixture/main.go` uses `transcriber.New(...)` and `transcript.Segment`
- [x] Obsolete types (`Segment`, `Word`) and functions (`ValidateSegmentTiming`, `RescaleWords`, `SplitSegments`) removed from `internal/transcriber`
- [x] `./test.sh` passes across all server packages
- [x] Zero backward-compatibility type aliases or legacy shims remain

## Blocked by

- #53 — Decouple domain consumers (Job, VFX Job, HTTP Handler)
