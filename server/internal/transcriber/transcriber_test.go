package transcriber

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/rubichandrap/subvision/server/internal/transcript"
)

func TestWordsFromTokens(t *testing.T) {
	cases := []struct {
		name    string
		seg     whisper.Segment
		want    []transcript.Word
		wantErr string
	}{
		{
			name: "groups continuation tokens and punctuation into words",
			seg: whisper.Segment{
				Start: 10 * time.Second,
				End:   12 * time.Second,
				Text:  "Hello, world!",
				Tokens: []whisper.Token{
					{Text: " Hello", Start: 10 * time.Second, End: 10600 * time.Millisecond},
					{Text: ",", Start: 10600 * time.Millisecond, End: 10640 * time.Millisecond},
					{Text: " world", Start: 10700 * time.Millisecond, End: 12 * time.Second},
					{Text: "!", Start: 12 * time.Second, End: 12 * time.Second},
				},
			},
			want: []transcript.Word{
				{Text: "Hello,", Start: 10, End: 10.64},
				{Text: "world!", Start: 10.7, End: 12},
			},
		},
		{
			name: "clamps token times into the segment window",
			seg: whisper.Segment{
				Start: 5 * time.Second,
				End:   6 * time.Second,
				Text:  "hi",
				Tokens: []whisper.Token{
					{Text: " hi", Start: 4900 * time.Millisecond, End: 61 * time.Second},
				},
			},
			want: []transcript.Word{{Text: "hi", Start: 5, End: 6}},
		},
		{
			name: "skips special and empty tokens",
			seg: whisper.Segment{
				Start: 1 * time.Second,
				End:   2 * time.Second,
				Text:  "go",
				Tokens: []whisper.Token{
					{Text: "[_BEG_]", Start: 0, End: 0},
					{Text: " ", Start: 0, End: 0},
					{Text: " go", Start: 1 * time.Second, End: 1500 * time.Millisecond},
					{Text: "[_TT_5]", Start: 0, End: 0},
				},
			},
			want: []transcript.Word{{Text: "go", Start: 1, End: 1.5}},
		},
		{
			name: "errors when token timestamps are missing",
			seg: whisper.Segment{
				Start: 3 * time.Second,
				End:   4 * time.Second,
				Text:  "silent timestamps",
				Tokens: []whisper.Token{
					{Text: " silent", Start: 0, End: 0},
					{Text: " timestamps", Start: 0, End: 0},
				},
			},
			wantErr: "no token timestamps",
		},
		{
			name: "a segment without text tokens yields no words",
			seg: whisper.Segment{
				Start: 1 * time.Second,
				End:   2 * time.Second,
				Tokens: []whisper.Token{
					{Text: "[_BEG_]", Start: 0, End: 0},
				},
			},
			want: []transcript.Word{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := wordsFromTokens(tc.seg)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("wordsFromTokens() error = %v, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("wordsFromTokens() unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("words = %+v, want %+v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("words[%d] = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestDefaultWordThresholdsMatchUpstream(t *testing.T) {
	// Upstream whisper.cpp defaults: thold_pt and thold_ptsum are both
	// 0.01f (src/whisper.cpp, --word-thold CLI flag).
	if DefaultTokenThreshold != 0.01 {
		t.Errorf("DefaultTokenThreshold = %v, want 0.01", DefaultTokenThreshold)
	}
	if DefaultTokenSumThreshold != 0.01 {
		t.Errorf("DefaultTokenSumThreshold = %v, want 0.01", DefaultTokenSumThreshold)
	}
}

type fakeThresholdContext struct {
	token, tokenSum float32
}

func (f *fakeThresholdContext) SetTokenThreshold(t float32)    { f.token = t }
func (f *fakeThresholdContext) SetTokenSumThreshold(t float32) { f.tokenSum = t }

func TestApplyWordThresholdsReachesDecoderContext(t *testing.T) {
	fake := &fakeThresholdContext{}
	applyWordThresholds(fake, 0.05, 0.07)
	if fake.token != 0.05 || fake.tokenSum != 0.07 {
		t.Errorf("thresholds = (%v, %v), want (0.05, 0.07)", fake.token, fake.tokenSum)
	}

	defaults := &fakeThresholdContext{}
	applyWordThresholds(defaults, DefaultTokenThreshold, DefaultTokenSumThreshold)
	if defaults.token != 0.01 || defaults.tokenSum != 0.01 {
		t.Errorf("defaults = (%v, %v), want (0.01, 0.01)", defaults.token, defaults.tokenSum)
	}
}

func TestNew(t *testing.T) {
	settings := Settings{ModelPath: "models/whisper.bin"}
	tr := New(settings)
	if tr == nil {
		t.Fatal("New() returned nil")
	}
	if tr.settings != settings {
		t.Errorf("tr.settings = %+v, want %+v", tr.settings, settings)
	}
}

func TestTranscriberTranscribe(t *testing.T) {
	tr := New(Settings{ModelPath: "unused"})

	t.Run("fails when wav cannot be loaded", func(t *testing.T) {
		_, err := tr.Transcribe("nonexistent.wav")
		if err == nil || !strings.Contains(err.Error(), "failed to load wav") {
			t.Fatalf("tr.Transcribe() error = %v, want 'failed to load wav'", err)
		}
	})

	t.Run("fails when model cannot be loaded", func(t *testing.T) {
		var segs []transcript.Segment
		var err error
		segs, err = tr.Transcribe(writeTestWav(t, 1))
		if err == nil || !strings.Contains(err.Error(), "failed to load whisper model") {
			t.Fatalf("tr.Transcribe() error = %v, want 'failed to load whisper model'", err)
		}
		if segs != nil {
			t.Errorf("expected nil segments on error, got %v", segs)
		}
	})
}


// writeTestWav writes a valid mono 16-bit PCM wav of silent audio at the
// whisper sample rate, so the wiring tests exercise the real wav loader.
func writeTestWav(t *testing.T, seconds float64) string {
	t.Helper()
	n := int(seconds * whisper.SampleRate)
	data := make([]byte, 44+n*2)
	copy(data[0:], "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(36+n*2))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], whisper.SampleRate)
	binary.LittleEndian.PutUint32(data[28:], whisper.SampleRate*2)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(n*2))

	path := filepath.Join(t.TempDir(), "audio.wav")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write test wav: %v", err)
	}
	return path
}
