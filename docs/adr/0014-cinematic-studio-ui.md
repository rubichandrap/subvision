# ADR-0014: Cinematic studio UI replaces the neobrutalist chrome

Date: 2026-11-16 · Status: accepted · Supersedes: ADR-0009

## Context

ADR-0009 gave the client a full neobrutalist chrome: 2px ink borders, hard offset shadows, a yellow/pink/cyan palette, uppercase mono labels. The request now is the opposite register: a modern, cinematic, editorial experience of the kind seen on Awwwards-style sites, responsive on phone and tablet as well as desktop.

## Decision

- **Scope** is the chrome only. The Subtitle Style (the burned-in render, part of the Edit Spec) and the frame preview that shows it are untouched; footage stays on neutral black.
- **Direction:** a dark-first "studio". Deep ink background with a slow drifting violet/cyan aurora, film grain and a vignette; cool paper in light mode. One indigo accent, used as a brand gradient on the primary action and for emphasis, never as decoration. Hairline borders, generous radii (`rounded-2xl` and pill buttons), soft elevation (`shadow-soft*`), frosted glass for floating chrome. Geist and Geist Mono replace Space Grotesk, Inter and Space Mono; display type is large and tightly tracked, labels are sentence case, not uppercase.
- **Home** is an editorial page: oversized headline, a live caption stage that demonstrates the product (a letterboxed frame with word-by-word karaoke captions over a ticking timecode), a marquee, a full-page drop target, and three numbered steps with pointer-tracked spotlights. The footer carries a giant outlined wordmark.
- **Motion** explains or rewards: line-mask headline reveal, scroll reveals, the caption demo, hover spotlights. It is CSS plus two small `IntersectionObserver` uses, with no new dependency. Under `prefers-reduced-motion` the CSS animations stop, scroll reveals show content immediately and the caption demo holds a finished phrase; the demo also pauses off screen and is hidden from assistive technology. Content is visible by default: only blocks that start below the fold are armed to animate in, and a timeout reveals them if the observer never fires.
- **Responsive** is structural. Navigation is a floating pill that drops its labels and collapses "New video" to an icon on phones. The editor stacks preview, trim and panel on small screens, panel height is capped only at `lg`, and "Generate subtitles" is pinned to the bottom edge below `lg`. Toasts appear top-right below `lg` so they never cover the pinned action bar. The gallery filter row scrolls horizontally, card actions that were hover-only are always visible on touch, and tap targets on touch layouts are at least 40 px (the frame preview's resize handle is footage chrome and was left as it was), and grid tracks use `minmax(0, 1fr)` so long file names truncate instead of widening the page.
- **Tokens:** `--primary` is indigo; `bg-accent` and `bg-secondary` are now neutral surfaces, not cyan and pink. `shadow-brutal*` utilities are gone; use `shadow-soft*`. Application components reference theme tokens, not raw hex; the decorative ambient layer and the caption demo (which imitates footage) use literal colours on purpose. Status colours are `success`, `warning`, `destructive`.

## Alternatives considered

- **Soften ADR-0009 (keep thick borders, round the corners).** Reads as the old theme. Rejected.
- **A motion library (Framer Motion, GSAP) and WebGL.** More spectacle, but a new dependency, a larger bundle and fragile reduced-motion behaviour for what CSS achieves here. Rejected for now.
- **Light-first.** The product is about video, which sits better on a dark ground; light mode is kept and works.

## Consequences

- Every `components/ui/*` primitive changed shape. ADR-0009's tokens no longer apply.
- Fonts are fetched from Google Fonts at build time by `next/font`, as before.
- The caption demo is illustrative only; it makes no claim about timing or accuracy.
- The editor and gallery were verified in headless Chromium at 1440, 820 and 390 px in both themes, with no horizontal overflow and no console errors. The editor hand-off in `lib/pending-video.ts` still breaks under React StrictMode in `next dev` (a second effect run finds the video already taken); production builds are unaffected, and it was left alone here.
