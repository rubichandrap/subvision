package transcriber

import (
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

func TestWordsFromTokens(t *testing.T) {
	cases := []struct {
		name    string
		seg     whisper.Segment
		want    []Word
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
			want: []Word{
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
			want: []Word{{Text: "hi", Start: 5, End: 6}},
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
			want: []Word{{Text: "go", Start: 1, End: 1.5}},
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
			want: []Word{},
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

func TestParseSilencedetect(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   []silenceInterval
	}{
		{
			name: "parses start and end pairs",
			output: "[silencedetect @ 0x55f] silence_start: 3.103\n" +
				"[silencedetect @ 0x55f] silence_end: 13.5 | silence_duration: 10.397",
			want: []silenceInterval{{3.103, 13.5}},
		},
		{
			name:   "a dangling start means the audio ends inside the silence",
			output: "[silencedetect @ 0x55f] silence_start: 20.0\n",
			want:   []silenceInterval{{20, math.Inf(1)}},
		},
		{
			name:   "multiple intervals come out in order",
			output: "silence_start: 1\nsilence_end: 2 | silence_duration: 1\nnoise\nsilence_start: 5\nsilence_end: 6 | silence_duration: 1\n",
			want:   []silenceInterval{{1, 2}, {5, 6}},
		},
		{
			name:   "unrelated output yields no silences",
			output: "Input #0, wav, 'audio.wav':\n  Duration: 00:00:24.00\n",
			want:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSilencedetect(tc.output)
			if len(got) != len(tc.want) {
				t.Fatalf("parseSilencedetect() = %+v, want %+v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("silences[%d] = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestSpeechWindows(t *testing.T) {
	cases := []struct {
		name     string
		audioDur float64
		silences []silenceInterval
		want     []speechWindow
	}{
		{
			name:     "no silence: one window over the whole audio",
			audioDur: 10,
			silences: nil,
			want:     []speechWindow{{0, 10}},
		},
		{
			name:     "leading and trailing silence are dropped",
			audioDur: 10,
			silences: []silenceInterval{{0, 2}, {8, 10}},
			want:     []speechWindow{{2, 8}},
		},
		{
			name:     "silences shorter than the split threshold stay inside the window",
			audioDur: 10,
			silences: []silenceInterval{{2, 3}, {6, 7}},
			want:     []speechWindow{{0, 10}},
		},
		{
			name:     "only silences at or past the split threshold become boundaries",
			audioDur: 10,
			silences: []silenceInterval{{0, 2.5}, {4, 4.8}, {6, 9}},
			want:     []speechWindow{{2.5, 6}, {9, 10}},
		},
		{
			name:     "all silence: no windows",
			audioDur: 10,
			silences: []silenceInterval{{0, math.Inf(1)}},
			want:     nil,
		},
		{
			name:     "speech blips below the minimum window are dropped",
			audioDur: 10,
			silences: []silenceInterval{{0, 4.95}, {5.05, 10}},
			want:     nil,
		},
		{
			name:     "silences are clamped, sorted, and merged",
			audioDur: 10,
			silences: []silenceInterval{{5, 9}, {-2, 5.5}, {9.5, 20}},
			// {5,9} overlaps {0,5.5} and merges into it: [0,9] is all silence;
			// {9.5,20} clamps to a 0.5 s silence — below the split threshold.
			want: []speechWindow{{9, 10}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := speechWindows(tc.audioDur, tc.silences)
			if len(got) != len(tc.want) {
				t.Fatalf("speechWindows() = %+v, want %+v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("windows[%d] = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestClampSegmentsToWindow(t *testing.T) {
	w := speechWindow{start: 3, end: 10}
	got := clampSegmentsToWindow([]Segment{
		{
			Start: 3.3, End: 12, Text: "hello world",
			Words: []Word{
				{Text: "hello", Start: 3.3, End: 4},
				{Text: "world", Start: 9.5, End: 12},
			},
		},
		{
			Start: 10.5, End: 11.5, Text: "leak",
			Words: []Word{{Text: "leak", Start: 10.5, End: 11.5}},
		},
		{
			Start: 2, End: 3.2, Text: "early",
			Words: []Word{{Text: "early", Start: 2, End: 3.2}},
		},
		{
			Start: 1, End: 1.5, Text: "wholly before",
			Words: []Word{{Text: "wholly before", Start: 1, End: 1.5}},
		},
	}, w)

	want := []Segment{
		{
			Start: 3.3, End: 10, Text: "hello world",
			Words: []Word{
				{Text: "hello", Start: 3.3, End: 4},
				{Text: "world", Start: 9.5, End: 10},
			},
		},
		{
			Start: 3, End: 3.2, Text: "early",
			Words: []Word{{Text: "early", Start: 3, End: 3.2}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("clampSegmentsToWindow() = %+v, want %+v", got, want)
	}
}

func TestNoSpeechDetected(t *testing.T) {
	cases := []struct {
		name     string
		settings Settings
		windows  []speechWindow
		want     bool
	}{
		{"gating off: windows or not, no finding", Settings{}, nil, false},
		{"gating off with windows", Settings{}, []speechWindow{{0, 1}}, false},
		{"gating on with zero windows means no speech", Settings{SpeechGating: true}, nil, true},
		{"gating on with windows", Settings{SpeechGating: true}, []speechWindow{{0, 1}}, false},
	}
	for _, tc := range cases {
		if got := noSpeechDetected(tc.settings, tc.windows); got != tc.want {
			t.Errorf("%s: noSpeechDetected() = %v, want %v", tc.name, got, tc.want)
		}
	}
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

// silenceDetectionError is a sentinel so the wiring test can assert the
// detection failure propagates unwrapped.
type silenceDetectionError struct{}

func (*silenceDetectionError) Error() string { return "silence detection failed" }

func TestTranscribeWiring(t *testing.T) {
	realDetect := detectSilences
	t.Cleanup(func() { detectSilences = realDetect })

	t.Run("gating off never runs silence detection", func(t *testing.T) {
		detectSilences = func(string) ([]silenceInterval, error) {
			t.Error("silence detection ran with gating disabled")
			return nil, nil
		}
		_, err := Transcribe(Settings{ModelPath: "unused"}, writeTestWav(t, 1))
		if err == nil || !strings.Contains(err.Error(), "failed to load whisper model") {
			t.Fatalf("Transcribe() error = %v, want it to contain %q", err, "failed to load whisper model")
		}
	})

	t.Run("gating on with no detected speech transcribes empty without the model", func(t *testing.T) {
		detectSilences = func(string) ([]silenceInterval, error) {
			return []silenceInterval{{0, math.Inf(1)}}, nil
		}
		segments, err := Transcribe(Settings{ModelPath: "unused", SpeechGating: true}, writeTestWav(t, 5))
		if err != nil {
			t.Fatalf("Transcribe() error = %v, want a silent wav to transcribe empty", err)
		}
		if len(segments) != 0 {
			t.Errorf("Transcribe() = %+v, want no segments", segments)
		}
	})

	t.Run("gating on with speech proceeds to model load", func(t *testing.T) {
		detectSilences = func(string) ([]silenceInterval, error) { return nil, nil }
		_, err := Transcribe(Settings{ModelPath: "unused", SpeechGating: true}, writeTestWav(t, 1))
		if err == nil || !strings.Contains(err.Error(), "failed to load whisper model") {
			t.Fatalf("Transcribe() error = %v, want it to contain %q", err, "failed to load whisper model")
		}
	})

	t.Run("gating on with a failing detection fails loudly", func(t *testing.T) {
		detectSilences = func(string) ([]silenceInterval, error) {
			return nil, &silenceDetectionError{}
		}
		_, err := Transcribe(Settings{ModelPath: "unused", SpeechGating: true}, writeTestWav(t, 1))
		var want *silenceDetectionError
		if !errors.As(err, &want) {
			t.Fatalf("Transcribe() error = %v, want the detection failure propagated", err)
		}
	})
}
