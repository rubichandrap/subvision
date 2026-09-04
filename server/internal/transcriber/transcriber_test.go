package transcriber

import (
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
