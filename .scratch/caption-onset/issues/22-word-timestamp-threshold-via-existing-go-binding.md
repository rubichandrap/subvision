# 22 — word timestamp threshold via existing Go binding

**GitHub issue:** #22

**Status:** done (closed 2026-09-05)

## Parent

#20 (see `../spec.md`)

## What to build

The server transcriber names the word timestamp probability thresholds as
constants defaulting to the upstream defaults, wired through the existing Go
binding setters for token threshold and token sum threshold. Default behavior
is unchanged; values move only on measured evidence from the fixture video.
No VAD, no max-initial-timestamp change, no model swap.

## Acceptance criteria

- [x] Thresholds are named constants defaulting to upstream defaults (0.01)
- [x] Custom values reach the decoder context through the binding setters
- [x] Existing missing-timestamp error path still triggers when no token
      carries timestamps
- [x] Transcriber test suite extended with threshold wiring cases, all
      passing

## Done

Commit d2105b0 (amended from af2a029 — the amend only dropped the unused
`SetTokenTimestamps` from the `thresholdSetter` interface; stub cleanup
finished in 04abb8b). `DefaultTokenThreshold` / `DefaultTokenSumThreshold`
match the vendored upstream (whisper.cpp `thold_pt` / `thold_ptsum`, both
0.01f). Suite: `go vet` + `go test ./internal/...` green.
