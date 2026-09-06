# 01 — editor-styling

**GitHub issue:** #24

**Status:** done (closed 2026-09-06; implementation record — the work shipped
before this tracker entry existed)

## Parent

`../spec.md` (settled spec, grilling session 2026-09-03)

## Scope

The editor end to end: flow (dropzone → `/editor` → Generate → tus upload
carrying the Edit Spec → VFX Job → `/processes`), Frame crop-to-fill with
presets + Free drag, dual-handle Trim with looped preview, the four
Animations (fade, slide, karaoke, Pop) with client-side `random` resolution,
the Subtitle Style knobs with live preview, the mirrored Edit Spec contract
(`server/internal/editspec` ↔ `vfx/src/contract.ts`), and the shadcn /
Tailwind v4 dark-first studio UI.

## Acceptance criteria (as shipped)

- [x] Edit Spec travels as tus `editSpec` metadata, validated fail-loud in
      both mirrors (ADR-0003)
- [x] vfx applies the Edit Spec at render; `RENDER_TEMPLATE` remains the
      fallback for jobs without one
- [x] Frame crop-to-fill via ffmpeg `scale→crop`; overlay renders at the
      target frame size
- [x] Trim shifts segment times and cuts the video; transcription covers
      only the trim window
- [x] All four animations selectable; `random` resolved client-side to one
      concrete animation
- [x] Subtitle Style knobs with live editor preview; wordsPerPage slider end
      to end (follow-up 835ab16)
- [x] Studio UI on shadcn + Tailwind v4; gallery grid with hover previews

## Done

Commits 018fdb8 (ADR-0003 + glossary), ac1512f (Edit Spec contract),
98ba615 (vfx applies the Edit Spec), 7be5e01 (editor + studio + gallery,
PR #9), 812d82e (trim-window transcription), 835ab16 (wordsPerPage),
5605b87 (Style tab scroll fix). Filed and closed as #24 for the record.
