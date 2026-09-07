Promoted: #35

# 04 — Save edits and re-render

**What to build:** Saving writes the edited segments; a Re-render action publishes a fresh render job with the edited segments plus the original Edit Spec, moves the process back through rendering to done, and the same download link serves the corrected video (overwrite in place, no versioning).

**Blocked by:** 01 — Persist Transcription Segments per Upload; 03 — Transcript card on the Process detail page.

**Status:** ready-for-agent

- [ ] Save persists edits; reloading the detail page shows them
- [ ] Re-render moves done to rendering and back to done, with the status API reflecting each step
- [ ] Output carries edited text and timing with the original trim, frame, animation, and style
- [ ] Failed re-render surfaces its reason and never strands the process
- [ ] Re-render without edits behaves like the original render
