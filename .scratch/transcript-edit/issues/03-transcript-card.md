Promoted: #34

# 03 — Transcript card on the Process detail page

**What to build:** The Process detail page gains a Transcript card between the status section and the video: every segment with its text and start/end time. While transcribing it shows a loading state, never half-decoded text. Once rendering or done, text and per-segment timing are editable, with invalid timing (negative times, end before start, unordered segments) rejected in the form. It rides the existing per-process poll — no new polling mechanism.

**Blocked by:** 02 — Read segments endpoint.

**Status:** ready-for-agent

- [ ] Card visible on the detail page for rendering and done processes
- [ ] Transcribing shows loading, never partial text
- [ ] Text and per-segment start/end editable with client-side validation messages
- [ ] No new polling; existing per-process poll drives updates
