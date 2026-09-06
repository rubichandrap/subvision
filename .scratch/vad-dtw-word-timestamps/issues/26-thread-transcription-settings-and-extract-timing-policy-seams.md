# 26 — Thread transcription settings and extract timing-policy seams

**What to build:** The transcriber receives its transcription settings — the
whisper model path and the VAD model path — threaded from server config as
parameters, and the timing policies (whether VAD is required, which DTW preset
matches a model) exist as pure, fake-testable decisions. No behavior changes in
this ticket: with nothing configured, transcription is exactly today's
behavior, so the seam the VAD and DTW tickets plug into exists and is tested
before either mechanism lands.

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent (GitHub: #26, sub-issue of #25)

- [ ] Transcription settings (model path, VAD path) flow from server config
      into the transcriber as parameters; nothing new is read from the
      environment inside the transcriber.
- [ ] The VAD-required policy and the model-to-DTW-preset mapping are pure
      functions with unit tests (known model resolves its preset; an unknown
      model resolves none).
- [ ] No behavior change: server and vfx test suites pass unchanged and
      transcription output is unchanged.
- [ ] The existing threshold-wiring tests keep passing untouched.
