# ADR-0009: Neobrutalist chrome UI, dual light/dark modes

Date: 2026-09-07 · Status: accepted

## Context

The client chrome (header, hero, dropzone, editor panels, gallery) used a
soft dark-first theme: thin borders, blurred shadows, large radii, green
primary. The request: full neobrutalism across the chrome, with both light
and dark modes kept working.

## Decision

- Scope is chrome UI only. Subtitle Style (the burned-in render output, part
  of the Edit Spec) is untouched.
- Brutal primitives: 2px solid borders, solid offset shadows
  (`2px/4px/6px`, zero blur), radius 0 on button/badge/input/tabs,
  `rounded-lg` max on cards/dropzone. Hover lifts `-2px`, active presses
  `+2px` with shadow collapse. Focus is a 3px ring. Disabled drops shadow.
- Palette: light paper `#FFF9EC` with ink `#141414`; dark warm espresso
  `#2B241C` (never near-black) with paper borders, card `#3A3128`,
  muted `#4A4034`. Same accents both modes: primary
  yellow `#FFC900`, pink `#FF90E8`, blue `#90E8FF`, success green `#7DF29A`,
  red `#FF5C5C`. Status reads through the StageBadge color, not border
  opacity.
- Active filter-tab count chip is `bg-background text-foreground`: on dark
  espresso the chip inverts against the yellow tab so the number stays
  readable (a bare `bg-background` chip inherits no text color and the
  number vanishes).
- Type: Space Grotesk 700-800 headings, Inter body, Space Mono for
  badges/labels/numbers. Uppercase reserved for eyebrows and small labels.
- Video preview surfaces stay neutral black; brutal chrome never touches
  footage.

## Alternatives considered

- **Keep green primary** — safest identity, but the old palette cannot carry
  brutal contrast at 2px borders. Rejected in favor of a free palette.
- **Soft brutal (keep radii, small shadows)** — half-measure; reads as the
  old theme with thicker borders. Rejected; the ask was full neubrutalism.

## Consequences

- Every `components/ui/*` primitive changed shape; any future component must
  use the brutal tokens (`shadow-brutal*`, 2px borders) or it will look off.
- Border color flips per mode (ink on paper, paper on ink); components must
  reference `border-border`, never a hardcoded ink/paper.
