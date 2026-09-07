# ADR-0008: Decoder tuning for Timing Drift

Timing Drift (a Timed Word appearing before it is spoken, or late by a fraction of a second) is attacked one decoder knob at a time, each measured on the acceptance fixture and the reporter's ear check — never all at once, so every change has a known cause. Order: beam size 5 → 8, then entropy threshold 2.40 → 2.0, then the small.en model opt-in via WHISPER_MODEL_PATH, then token-probability logging. Token probabilities are logged, never used to drop words: dropping a low-confidence word turns drift into a missing word, which is worse. SetSplitOnWord was considered and rejected — it only moves segment cut points and we never set a max segment length, so it changes nothing. DTW and built-in VAD stay out per ADR-0007 (no vendored edits). Each step reverts if the fixture does not move.

## Measured 2026-09-07: all four decoder-side knobs failed

Beam 8, entropy 2.0, and small.en each tested on the reporter's video with no significant change — Timing Drift persists. Decoder tuning is exhausted at the stock binding's surface. The remaining route is render-side compensation (e.g. shifting Caption Page timing against measured drift) or user-editable timing export (JSON/SRT), not more decode knobs.
