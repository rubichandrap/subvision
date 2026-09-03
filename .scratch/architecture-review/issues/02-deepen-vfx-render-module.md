# 02 — deepen-vfx-render-module

**GitHub issue:** #3

**Status:** ready-for-agent

## Parent

#1

## What to build

The vfx service's render path becomes one module that owns everything render-related: id derivation from the object key, temporary paths (with real file names and extensions), render options, the ffmpeg invocation, and the Output upload. A received VFX Job either produces an Output at `outputs/<id>` and publishes a JobCompleted event (upload id + output key), or is requeued; after repeated failures it dead-letters. The consumer can no longer wedge permanently after a few failures.

## Acceptance criteria

- [ ] A synthetic VFX Job results in a rendered Output object at `outputs/<id>` in object storage
- [ ] No caller passes directory paths where file paths are required; the module derives every path it uses
- [ ] Render options (frame rate, dimensions) come from configuration, not hardcoded call sites
- [ ] Failed jobs are requeued; repeated failures dead-letter after a bounded number of attempts
- [ ] A successful render publishes a JobCompleted event carrying the upload id and output key
- [ ] Consumer prefetch is configuration rather than a machine-specific constant

## Blocked by

- #2 — the job contract must exist before the render module consumes it.
