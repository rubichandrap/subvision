package transcriber

// Settings carries the transcription inputs threaded from server config: the
// whisper model to transcribe with and, optionally, speech gating before
// decoding (ADR-0007). Gating off keeps the pre-gating behavior: the whole
// audio is decoded in one pass.
type Settings struct {
	ModelPath    string
	SpeechGating bool
}

// GatingEnabled reports whether these settings gate decoding on detected
// speech: ffmpeg silencedetect finds the silence, the transcriber derives the
// speech windows, and only those windows are decoded (ADR-0007).
func (s Settings) GatingEnabled() bool {
	return s.SpeechGating
}
