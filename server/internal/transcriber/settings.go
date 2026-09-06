package transcriber

import (
	"path/filepath"
	"strings"
)

// Settings carries the transcription inputs threaded from server config:
// the whisper model to transcribe with and, optionally, the Silero VAD
// model that gates decoding on speech (ADR-0007). An empty VADModelPath
// keeps VAD off — the pre-ADR-0007 behavior.
type Settings struct {
	ModelPath    string
	VADModelPath string
}

// VADRequired reports whether these settings ask for VAD-gated
// transcription: a configured VAD model path means VAD is required, an
// empty one means the VAD-free path.
func (s Settings) VADRequired() bool {
	return s.VADModelPath != ""
}

// DTWPreset names a whisper.cpp alignment-heads preset (ADR-0007). The
// values mirror the vendored WHISPER_AHEADS_* enum, which the binding shim
// switches on.
type DTWPreset string

// dtwPresets maps a whisper model file's name (ggml- prefix and .bin suffix
// stripped) to its upstream alignment-heads preset. Bare "large" is
// deliberately absent: which large variant it means is a guess, and an
// unmatched model means DTW stays off.
var dtwPresets = map[string]DTWPreset{
	"tiny.en":        "tiny_en",
	"tiny":           "tiny",
	"base.en":        "base_en",
	"base":           "base",
	"small.en":       "small_en",
	"small":          "small",
	"medium.en":      "medium_en",
	"medium":         "medium",
	"large-v1":       "large_v1",
	"large-v2":       "large_v2",
	"large-v3":       "large_v3",
	"large-v3-turbo": "large_v3_turbo",
}

// DTWPresetFor resolves the alignment-heads preset matching a whisper model
// path by file name (ggml-base.en.bin → base.en). ok=false for models with
// no upstream preset: the caller transcribes without DTW and warns.
func DTWPresetFor(modelPath string) (DTWPreset, bool) {
	name := strings.TrimPrefix(strings.TrimSuffix(filepath.Base(modelPath), ".bin"), "ggml-")
	preset, ok := dtwPresets[name]
	return preset, ok
}
