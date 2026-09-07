Promoted: #52

## Parent

Part of #50

## What to build

Refactor `internal/transcriber` into a deep audio-decoding adapter. Define a `Transcriber` struct with constructor `New(settings Settings) *Transcriber` and method `(t *Transcriber) Transcribe(audioPath string) ([]transcript.Segment, error)`. Encapsulate whisper CGO context initialization, decoder thresholds, token decoding, and token-to-word translation entirely within the adapter. Decoded segments are run through `transcript.Split` before returning.

## Acceptance criteria

- [ ] `transcriber.New(settings Settings) *Transcriber` constructor defined
- [ ] `(t *Transcriber) Transcribe(audioPath string) ([]transcript.Segment, error)` returns pure domain segments
- [ ] Decoded segments are processed with `transcript.Split` to enforce speech pause bounds
- [ ] Whisper CGO bindings, model loading, and token grouping are encapsulated inside `transcriber`
- [ ] Existing transcriber wiring tests and settings tests pass

## Blocked by

- #51 — Pure domain transcript module and timing invariants
