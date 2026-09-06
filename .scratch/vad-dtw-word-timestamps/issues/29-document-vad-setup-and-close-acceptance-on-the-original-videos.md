# 29 — Document VAD setup and close acceptance on the original videos

**What to build:** Setup documentation ships with the feature: the server env
example and the README whisper setup gain the VAD model path, its Silero
download step, and the unset/set semantics (legacy behavior vs VAD required,
fail-fast and empty-transcription policies). Final acceptance then closes the
parent spec: the final measured numbers are posted, and the reporter's ear
check on the original videos confirms text no longer appears before speech —
at video start or mid-video.

**Blocked by:** #27 (VAD) and #28 (DTW) — final numbers and docs must describe
the shipped combination.

**Status:** ready-for-agent (GitHub: #29, sub-issue of #25)

- [ ] The server env example gains the VAD model path var with a comment; the
      README whisper setup documents the Silero download step and the unset/set
      semantics.
- [ ] Final fixture numbers (VAD+DTW) are posted on the parent issue,
      including the VAD-only vs VAD+DTW comparison.
- [ ] Reporter ear check on the original videos passes: no caption visible
      before speech, anywhere.
- [ ] The parent issue is closed with the acceptance evidence.
