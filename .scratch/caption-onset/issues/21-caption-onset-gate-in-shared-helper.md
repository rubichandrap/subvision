# 21 — caption onset gate in shared helper

**GitHub issue:** #21

**Status:** done (closed 2026-09-05)

## Parent

#20 (see `../spec.md`)

## What to build

No word-driven caption appears before its first word starts. The shared
caption paging helper gains the onset rule, so karaoke, pop, fade, and slide
are all covered through the single seam they route through. Karaoke keeps
rendering the page it is given with no extra per-word filter; pop keeps its
existing per-word stagger filter inside a visible page; fade and slide keep
their fade curves but also respect the onset rule. Wordless segments render
exactly as before.

## Acceptance criteria

- [x] Time before a segment's first word start yields no visible caption words
- [x] Exactly at first word start the first page appears
- [x] Existing paging behavior unchanged: page switch on next page's first
      word, gap hold, trailing hold until segment end
- [x] Wordless segments pass through unaffected
- [x] Shared template test suite extended with pre-onset, onset-edge, and
      wordless cases, all passing

## Done

Commits 59b6cbb + d66caf0; review follow-up 841e416 folded the gate into
`activeSegment` (the single seam) and anchored the fade/slide entrances to
the onset. Test suite: pre-onset, onset-edge, and discriminating wordless
cases in `vfx/src/templates/shared.test.ts`.
