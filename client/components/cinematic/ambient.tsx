// Fixed atmosphere behind every page: drifting colour, film grain, vignette.
// Purely decorative, so it is hidden from assistive tech and never takes
// pointer events.
export function Ambient() {
  return (
    <div aria-hidden className="pointer-events-none fixed inset-0 -z-10 overflow-hidden">
      <div className="absolute -left-[10%] top-[-18%] h-[60vmax] w-[60vmax] rounded-full bg-[radial-gradient(closest-side,rgb(124_108_255/0.30),transparent)] opacity-70 blur-3xl motion-safe:animate-[drift-a_26s_ease-in-out_infinite] dark:opacity-90" />
      <div className="absolute -right-[12%] top-[10%] h-[50vmax] w-[50vmax] rounded-full bg-[radial-gradient(closest-side,rgb(34_211_238/0.20),transparent)] opacity-60 blur-3xl motion-safe:animate-[drift-b_32s_ease-in-out_infinite]" />
      <div className="absolute bottom-[-30%] left-[25%] h-[46vmax] w-[46vmax] rounded-full bg-[radial-gradient(closest-side,rgb(255_92_127/0.14),transparent)] opacity-60 blur-3xl motion-safe:animate-[drift-a_38s_ease-in-out_infinite_reverse]" />
      {/* Film grain */}
      <svg className="absolute inset-0 h-full w-full opacity-[0.07] mix-blend-overlay dark:opacity-[0.11]">
        <filter id="grain">
          <feTurbulence type="fractalNoise" baseFrequency="0.85" numOctaves="2" stitchTiles="stitch" />
          <feColorMatrix type="saturate" values="0" />
        </filter>
        <rect width="100%" height="100%" filter="url(#grain)" />
      </svg>
      {/* Vignette */}
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,transparent_55%,rgb(0_0_0/0.45))] dark:block hidden" />
    </div>
  );
}
