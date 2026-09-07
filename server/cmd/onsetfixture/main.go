// Command onsetfixture regenerates the caption onset acceptance fixture's
// segments.json (server/testdata/onset-fixture): it transcribes the fixture
// WAV through the production transcriber and writes the Transcription
// Segments the onset gate is verified against (ADR-0006, issue #23).
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/rubichandrap/subvision/server/internal/transcriber"
	"github.com/rubichandrap/subvision/server/internal/transcript"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatalf("usage: onsetfixture <model-path> <wav-path> <out-json-path>")
	}
	modelPath, wavPath, outPath := os.Args[1], os.Args[2], os.Args[3]

	trans := transcriber.New(transcriber.Settings{
		ModelPath: modelPath,
	})
	var segments []transcript.Segment
	var err error
	segments, err = trans.Transcribe(wavPath)
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
