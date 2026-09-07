Promoted: #36

# 01 — Drop SPEECH_GATING

**What to build:** Speech gating is removed entirely: no flag, no silence-detection subprocess, no windowed decode path. Every transcription decodes the full audio in one pass. The onset gate stays exactly as is; the acceptance fixture is re-recorded ungated; docs stop describing gating as a feature and the superseded ADR is marked as such.

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

- [ ] Single decode path; no gating flag or silence detection anywhere
- [ ] Onset gate still holds on the re-recorded fixture
- [ ] Gating tests removed; remaining suite passes
- [ ] No gating references left in README, env examples, glossary, or code comments
