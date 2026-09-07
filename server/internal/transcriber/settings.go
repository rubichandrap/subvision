package transcriber

// Settings carries the transcription inputs threaded from server config: the
// whisper model to transcribe with. Decoding is always one pass over the
// whole audio; the onset gate (ADR-0006) lives on the render side.
type Settings struct {
	ModelPath string
}
