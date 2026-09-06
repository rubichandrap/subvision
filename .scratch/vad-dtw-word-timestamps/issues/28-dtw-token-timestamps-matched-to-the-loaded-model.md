# 28 — DTW token timestamps matched to the loaded model

**What to build:** DTW token timestamps turn on at model load with the
alignment-heads preset matching the loaded model — the current base.en model
maps to its preset — so per-token timings align tightly to the audio frames
that produced them and within-speech drift shrinks. An unrecognized model falls
back to heuristic timestamps with a warning, never a failure. DTW is not
separately gated. The vendored Go binding's cgo shim is extended at the
model-load path to carry the DTW enable flag and preset.

**Blocked by:** #27 (fixture evidence must be measured with both mechanisms on;
both touch the same vendored shim).

**Status:** ready-for-agent (GitHub: #28, sub-issue of #25)

- [ ] With the configured model, model load enables DTW token timestamps with
      the matching alignment-heads preset; the vendored shim change is minimal
      and localized.
- [ ] An unrecognized model disables DTW with a warning and transcription
      still succeeds on heuristic timestamps.
- [ ] Unit tests cover preset resolution for the known model, fallback for an
      unknown one, and the wiring through the threaded settings.
- [ ] The onset fixture regen command runs with VAD+DTW enabled; the
      regenerated segments are recorded; mid-speech alignment is compared
      against the VAD-only numbers and the delta noted on the parent issue.
- [ ] The vfx fixture suite is green on the regenerated segments (the onset
      gate still locks).
