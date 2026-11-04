package asr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rubichandrap/subvision/server/internal/transcript"
)

func writeParakeetWav(t *testing.T) string {
	t.Helper()
	data := []byte{
		'R', 'I', 'F', 'F', 0x24, 0x00, 0x00, 0x00,
		'W', 'A', 'V', 'E', 'f', 'm', 't', ' ',
		0x10, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00,
		0x80, 0x3e, 0x00, 0x00, 0x00, 0x7d, 0x00, 0x00,
		0x02, 0x00, 0x10, 0x00, 'd', 'a', 't', 'a',
		0x00, 0x00, 0x00, 0x00,
	}
	path := filepath.Join(t.TempDir(), "audio.wav")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write parakeet wav: %v", err)
	}
	return path
}

func TestParakeetTranscriberTranscribe(t *testing.T) {
	segments := []transcript.Segment{{
		Start: 1,
		End:   2,
		Text:  "hello",
		Words: []transcript.Word{{Text: "hello", Start: 1, End: 2}},
	}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "audio/wav" {
			t.Errorf("Content-Type = %q, want audio/wav", got)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(parakeetResponse{Segments: segments}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	transcriber := NewParakeet(server.URL, time.Second)
	got, err := transcriber.Transcribe(writeParakeetWav(t))
	if err != nil {
		t.Fatalf("Transcribe() unexpected error: %v", err)
	}
	if len(got) != len(segments) {
		t.Fatalf("segments = %+v, want %+v", got, segments)
	}
	if got[0].Text != "hello" {
		t.Errorf("segment text = %q, want %q", got[0].Text, "hello")
	}
}

func TestParakeetTranscriberAcceptsContiguousWordBoundaries(t *testing.T) {
	segments := []transcript.Segment{{
		Start: 1,
		End:   3,
		Text:  "hello world",
		Words: []transcript.Word{
			{Text: "hello", Start: 1.0, End: 2.0},
			{Text: "world", Start: 2.0, End: 3.0},
		},
	}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(parakeetResponse{Segments: segments})
	}))
	defer server.Close()

	transcriber := NewParakeet(server.URL, time.Second)
	got, err := transcriber.Transcribe(writeParakeetWav(t))
	if err != nil {
		t.Fatalf("Transcribe() unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("segments = %+v, want 1 segment", got)
	}
}

func TestParakeetTranscriberRejectsMateriallyOverlappingWords(t *testing.T) {
	segments := []transcript.Segment{{
		Start: 1,
		End:   3,
		Text:  "hello world",
		Words: []transcript.Word{
			{Text: "hello", Start: 1.0, End: 2.0},
			{Text: "world", Start: 1.99, End: 3.0},
		},
	}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(parakeetResponse{Segments: segments})
	}))
	defer server.Close()

	transcriber := NewParakeet(server.URL, time.Second)
	_, err := transcriber.Transcribe(writeParakeetWav(t))
	if err == nil || !strings.Contains(err.Error(), "overlaps the previous word") {
		t.Fatalf("Transcribe() error = %v, want it to contain %q", err, "overlaps the previous word")
	}
}

func TestParakeetTranscriberFailsLoudly(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		wantErr string
	}{
		{
			name: "worker returns non-200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "model unavailable", http.StatusServiceUnavailable)
			},
			wantErr: "parakeet worker returned 503",
		},
		{
			name: "worker returns malformed json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("{"))
			},
			wantErr: "failed to decode parakeet response",
		},
		{
			name: "worker returns segment without words",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(parakeetResponse{
					Segments: []transcript.Segment{{Start: 1, End: 2, Text: "hello"}},
				})
			},
			wantErr: "has no timed words",
		},
		{
			name: "worker returns segment with inverted word timing",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(parakeetResponse{
					Segments: []transcript.Segment{{
						Start: 1,
						End:   2,
						Text:  "hello",
						Words: []transcript.Word{{Text: "hello", Start: 1.5, End: 1.2}},
					}},
				})
			},
			wantErr: "has end < start",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			transcriber := NewParakeet(server.URL, time.Second)
			_, err := transcriber.Transcribe(writeParakeetWav(t))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Transcribe() error = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestParakeetTranscriberRequiresURL(t *testing.T) {
	transcriber := NewParakeet("", time.Second)
	_, err := transcriber.Transcribe(writeParakeetWav(t))
	if err == nil || !strings.Contains(err.Error(), "worker URL is not configured") {
		t.Fatalf("Transcribe() error = %v, want worker URL configuration error", err)
	}
}
