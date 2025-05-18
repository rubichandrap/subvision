package transcriber

import (
	"fmt"
	"os"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/go-audio/wav"
)

type Segment struct {
	Start float64
	End   float64
	Text  string
}

// Transcribe baca wav, proses dengan whisper, dan return segments
func Transcribe(modelPath, audioPath string) ([]Segment, error) {
	model, err := whisper.New(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load whisper model: %w", err)
	}
	defer model.Close()

	// Load wav ke float32 slice
	data, err := loadWavToFloat32(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load wav: %w", err)
	}

	ctx, err := model.NewContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create whisper context: %w", err)
	}

	if err := ctx.Process(data, nil, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to process audio: %w", err)
	}

	var segments []Segment
	for {
		seg, err := ctx.NextSegment()
		if err != nil {
			break
		}
		segments = append(segments, Segment{
			Start: seg.Start.Seconds(),
			End:   seg.End.Seconds(),
			Text:  seg.Text,
		})
	}

	return segments, nil
}

// Helper load wav file jadi float32 slice
func loadWavToFloat32(path string) ([]float32, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	dec := wav.NewDecoder(fh)
	if !dec.IsValidFile() {
		return nil, fmt.Errorf("invalid wav file")
	}

	buf, err := dec.FullPCMBuffer()
	if err != nil {
		return nil, err
	}

	if dec.SampleRate != whisper.SampleRate {
		return nil, fmt.Errorf("unsupported sample rate: %d", dec.SampleRate)
	}
	if dec.NumChans != 1 {
		return nil, fmt.Errorf("unsupported number of channels: %d", dec.NumChans)
	}

	return buf.AsFloat32Buffer().Data, nil
}
