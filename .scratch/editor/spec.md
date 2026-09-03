# Editor + styling: settled spec (grilling session 2026-09-03)

Decisions from the interview, in force for this implementation. Glossary terms
live in `CONTEXT.md` (Edit Spec, Frame, Frame Preset, Subtitle Style); the
architecture decision is recorded in `docs/adr/0003-edit-spec-via-tus-metadata.md`.

## Flow

choose video (home dropzone) → `/editor` (Frame, Trim, Animation, Subtitle
Style) → Generate → tus upload carries the Edit Spec as `editSpec` metadata →
server validates fail-loud → VFX Job applies it at render → `/processes`
gallery tracks the Process.

## Frame (crop-to-fill)

Presets: 9:16, 4:5, 1:1, 16:9, Free (drag corner, ratio 0.5–2.0 in the UI,
0.3–3.5 in the contract). Zoom 1–5×, pan ±1 (normalized overflow). Applied by
ffmpeg `scale→crop` server-side; the subtitle overlay renders at the target
frame size.

## Trim

Dual-handle slider, looped preview inside the window. Server shifts segment
times by the trim start and cuts the video with ffmpeg. No filmstrip this pass.

## Animation

`fade`, `slide`, `karaoke`, plus new **Pop** (shorts-style word-by-word
captions: words pop in one at a time with a spring scale, active word
highlighted; word timing = segment duration weighted by word length — segments
carry no word timestamps). `random` is resolved client-side at submit; the
contract always carries a concrete animation. vfx honors per-job animation,
`RENDER_TEMPLATE` stays as the fallback for jobs without an Edit Spec.

## Subtitle Style (knobs, all with live preview in the editor)

fontFamily (bundled: Montserrat, Inter, Poppins, Oswald, Bebas Neue, Anton),
fontSizeScale (0.5–2), color, outlineWidth (0–32 px @1080 reference) +
outlineColor, bottomMargin (0–0.8 fraction of frame height), background
(none|box) + backgroundOpacity, uppercase, highlightColor (karaoke/pop).
Preview uses sample text — transcription happens after upload.

## Contract

`editSpec` is optional in the VFX Job payload (jobs without one render as
before); when present, every field is required and validated in both mirrors
(`server/internal/editspec` and `vfx/src/contract.ts`). The old reserved
top-level `animationType` is removed.

## UI

shadcn on Tailwind v4, dark-first studio theme (near-black neutrals, one
chartreuse accent, Space Grotesk display + Inter body). Gallery = grid of
hover-preview videos from the existing download URL, no server changes.
WebGPU: intentionally skipped.
