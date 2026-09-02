# ADR-0003: The Edit Spec travels as tus metadata and is applied at render time

Date: 2026-09-03 · Status: accepted

## Context

The client gained an editor (Frame, Trim, Animation, Subtitle Style — the Edit
Spec in `CONTEXT.md`), which raised two questions: where the configuration
rides to the backend, and where it is applied. Applying it in the browser
would mean transcoding the video client-side before upload; the render path
already downloads, probes, and re-encodes the video once, and the vfx
service's Remotion/ffmpeg pass is the single place that knows how to compose.

## Decision

- The Edit Spec is serialized to JSON by the client and sent as the
  `editSpec` tus metadata key on `POST /files`. tus-js-client base64-encodes
  metadata values and tusd decodes them, so the server reads plain JSON from
  the upload's metadata map. The uploaded file itself is never modified.
- The server validates the spec fail-loud (`internal/editspec` mirrors
  `vfx/src/contract.ts`, like the rest of the VFX Job contract) and forwards
  it through the upload job message into the VFX Job payload. An invalid spec
  fails the Process with the validation detail; a missing one is legal and
  renders as before (full video, `RENDER_TEMPLATE`, default style) so old
  clients keep working.
- The vfx service applies the spec during its single render pass: ffmpeg
  cuts the trim window and scales/crops the source to the target Frame
  (crop-to-fill), while the Remotion subtitle overlay is rendered at the
  frame's dimensions with the requested animation and Subtitle Style.
- The contract's reserved top-level `animationType` field is removed; the
  animation lives inside the Edit Spec.

## Alternatives considered

- **Client pre-renders the video (ffmpeg.wasm/WebCodecs)** — no contract
  change, but a second encode on the user's machine, minutes of browser work
  for large files, memory pressure on mobile, and a second source of truth
  for render parameters. Rejected.
- **REST endpoint for edit configuration (`POST /jobs/:id/config`)** — splits
  the job's definition across two channels (tus metadata + JSON API) and
  needs its own coupling with resumable uploads (config may arrive before,
  during, or after the bytes). Metadata is atomic with the upload itself.
  Rejected.

## Consequences

- The `Upload-Metadata` header grows by ~1 KB; tus handles this and the
  server already plumbs the full metadata map into the upload job message.
- Both contract mirrors must validate the same field set; the wire-shape
  tests on each side make a one-sided change loud.
- Trim and crop are ffmpeg filter parameters derived from the spec — no
  intermediate video is ever stored; the Output is the first and only encode.
- Frame dimensions depend on the source video (the long side never upscales
  past 1920), so the vfx service probes the video before rendering.
