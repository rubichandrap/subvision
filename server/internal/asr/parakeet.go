package asr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/rubichandrap/subvision/server/internal/transcript"
)

// DefaultParakeetTimeout is the loud-failure ceiling for one transcription
// request. It is intentionally generous: long videos need real decode time.
const DefaultParakeetTimeout = 60 * time.Second

// ParakeetTranscriber transcribes audio by posting a 16 kHz mono WAV to a
// small Python ASR worker and reading back Subvision's segment contract.
type ParakeetTranscriber struct {
	URL    string
	Client *http.Client
}

// NewParakeet constructs a ParakeetTranscriber with the supplied worker URL.
func NewParakeet(url string, timeout time.Duration) *ParakeetTranscriber {
	if timeout <= 0 {
		timeout = DefaultParakeetTimeout
	}
	return &ParakeetTranscriber{
		URL:    url,
		Client: &http.Client{Timeout: timeout},
	}
}

type parakeetResponse struct {
	Segments []transcript.Segment `json:"segments"`
}

// Transcribe posts the WAV at audioPath to the Parakeet worker and returns the
// worker's Subvision-shaped Segments. Any missing or malformed output is a
// loud transcription failure, never a silent fallback.
func (p *ParakeetTranscriber) Transcribe(audioPath string) ([]transcript.Segment, error) {
	if p == nil || p.URL == "" {
		return nil, fmt.Errorf("parakeet worker URL is not configured")
	}
	if p.Client == nil {
		p.Client = &http.Client{Timeout: DefaultParakeetTimeout}
	}

	data, err := os.ReadFile(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wav: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.URL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to build parakeet request: %w", err)
	}
	req.Header.Set("Content-Type", "audio/wav")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("parakeet worker unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf(
			"parakeet worker returned %s: %s",
			resp.Status,
			string(bytes.TrimSpace(body)),
		)
	}

	var payload parakeetResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode parakeet response: %w", err)
	}
	if err := validateParakeetSegments(payload.Segments); err != nil {
		return nil, err
	}
	return payload.Segments, nil
}

// Contiguous words sharing a boundary are normal for transducer timestamps;
// only materially overlapping words are rejected.
const wordBoundaryEpsilon = 0.001 // seconds

// validateParakeetSegments makes malformed worker output impossible to pass
// downstream: every word must carry a non-empty text and ordered timings, and
// every segment must carry finite, ordered times.
func validateParakeetSegments(segments []transcript.Segment) error {
	for i, segment := range segments {
		if !finite(segment.Start) || !finite(segment.End) {
			return fmt.Errorf("parakeet segment %d carries non-finite start or end", i+1)
		}
		if segment.End <= segment.Start {
			return fmt.Errorf("parakeet segment %d has end <= start", i+1)
		}
		if len(segment.Words) == 0 {
			return fmt.Errorf("parakeet segment %d has no timed words", i+1)
		}
		for j, word := range segment.Words {
			if word.Text == "" {
				return fmt.Errorf("parakeet segment %d word %d has empty text", i+1, j+1)
			}
			if !finite(word.Start) || !finite(word.End) {
				return fmt.Errorf("parakeet segment %d word %d has non-finite timing", i+1, j+1)
			}
			if word.End < word.Start {
				return fmt.Errorf("parakeet segment %d word %d has end < start", i+1, j+1)
			}
			if word.Start < segment.Start || word.End > segment.End {
				return fmt.Errorf("parakeet segment %d word %d lies outside the segment window", i+1, j+1)
			}
			if j > 0 && word.Start < segment.Words[j-1].End-wordBoundaryEpsilon {
				return fmt.Errorf("parakeet segment %d word %d overlaps the previous word", i+1, j+1)
			}
		}
	}
	return nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
