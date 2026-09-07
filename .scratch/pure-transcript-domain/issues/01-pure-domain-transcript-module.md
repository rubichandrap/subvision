Promoted: #51

## Parent

Part of #50

## What to build

Establish a pure Go `internal/transcript` domain package with zero external dependencies and zero CGO. Define `Segment` (Transcription Segment) and `Word` (Timed Word). Implement pure domain operations: `ValidateTiming` verifying non-negative, finite, positive-duration, and chronological bounds returning a typed `ValidationError`; `RescaleWords` proportionally scaling word timestamps into edited segment windows; and `Split` cutting segments on natural speech pauses (>= 0.4s) with maximum word and duration bounds adhering to ADR-0005.

## Acceptance criteria

- [x] `internal/transcript` package exists with zero CGO dependencies
- [x] `transcript.Segment` and `transcript.Word` structs defined matching domain specifications
- [x] `transcript.ValidateTiming` rejects non-finite values, negative starts, `end <= start`, and non-chronological overlap with a typed `ValidationError`
- [x] `transcript.RescaleWords` proportionally scales word timestamps into edited windows while preserving relative offsets
- [x] `transcript.Split` segments speech on pauses >= 0.4s and enforces max 12 words / 8.0s duration caps per ADR-0005
- [x] Pure unit tests in `internal/transcript` verify validation, rescaling, and pause splitting, passing with standard `go test` without whisper CGO flags

## Blocked by

- None — can start immediately.
