// Command onsetfixture regenerates the caption onset acceptance fixture's
// segments.json (server/testdata/onset-fixture): it transcribes the fixture
// WAV through the production transcriber and writes the Transcription
// Segments the onset gate is verified against (ADR-0006, issue #23).
//
// Speech gating comes from VAD_GATING — the same env the server reads
// (ADR-0007). Set it to regenerate the fixture gated on detected speech;
// unset regenerates the pre-gating behavior.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/transcriber"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatalf("usage: onsetfixture <model-path> <wav-path> <out-json-path>")
	}
	modelPath, wavPath, outPath := os.Args[1], os.Args[2], os.Args[3]

	segments, err := transcriber.Transcribe(transcriber.Settings{
		ModelPath: modelPath,
		VADGating: config.VADGatingFromEnv(),
	}, wavPath)
	if err != nil {
		log.Fatalf("transcribe: %v", err)
	}

	data, err := json.MarshalIndent(segments, "", "  ")
	if err != nil {
		log.Fatalf("marshal segments: %v", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		log.Fatalf("write %s: %v", outPath, err)
	}
	fmt.Printf("wrote %d segments to %s\n", len(segments), outPath)
}
