package transcriber

import (
	"fmt"
	"os"
	"strings"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/go-audio/wav"
)

type Word struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
	Words []Word  `json:"words"`
}

// transcribes the audio file at audioPath using the Whisper model at modelPath
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

	// Word timings are transcribed, not derived: token timestamps make
	// whisper time every token, which the words below are built from.
	ctx.SetTokenTimestamps(true)

	if err := ctx.Process(data, nil, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to process audio: %w", err)
	}

	var segments []Segment
	for n := 0; ; n++ {
		seg, err := ctx.NextSegment()
		if err != nil {
			break
		}
		words, err := wordsFromTokens(seg)
		if err != nil {
			return nil, fmt.Errorf("transcription segment %d: %w", n, err)
		}
		segments = append(segments, Segment{
			Start: seg.Start.Seconds(),
			End:   seg.End.Seconds(),
			Text:  seg.Text,
			Words: words,
		})
	}

	return segments, nil
}

// wordsFromTokens groups a segment's whisper tokens into Words carrying
// per-word timings. A token whose text begins with a space begins a new word
// (segment text is the plain concatenation of token texts); continuations and
// the punctuation that follows extend the current word. Token times are
// clamped into the segment's window.
//
// A segment that carries text but not a single non-zero token timestamp means
// token timestamps were never computed — that is an error, because word
// timings must come from whisper, they are never guessed.
func wordsFromTokens(seg whisper.Segment) ([]Word, error) {
	segStart := seg.Start.Seconds()
	segEnd := seg.End.Seconds()
	words := []Word{}
	seenText := false
	timestamped := false

	for _, token := range seg.Tokens {
		// "[_" is whisper.cpp's own special-token marker ([_BEG_], [_TT_2_], …).
		if strings.HasPrefix(token.Text, "[_") {
			continue
		}
		text := strings.TrimSpace(token.Text)
		if text == "" {
			continue
		}
		seenText = true
		if token.Start != 0 || token.End != 0 {
			timestamped = true
		}

		start := clampSeconds(token.Start.Seconds(), segStart, segEnd)
		end := clampSeconds(token.End.Seconds(), segStart, segEnd)
		if end < start {
			end = start
		}

		if strings.HasPrefix(token.Text, " ") || len(words) == 0 {
			words = append(words, Word{Text: text, Start: start, End: end})
		} else {
			last := &words[len(words)-1]
			last.Text += text
			last.End = end
		}
	}

	if seenText && !timestamped && seg.End > 0 {
		return nil, fmt.Errorf(
			"whisper returned no token timestamps (word timings unavailable); is token_timestamps enabled?")
	}
	return words, nil
}

func clampSeconds(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// loads a WAV file and returns its audio data as a slice of float32
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

