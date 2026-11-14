package sourcetimeline

import (
	"reflect"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/transcript"
)

func w(text string, start, end float64) transcript.Word {
	return transcript.Word{Text: text, Start: start, End: end}
}

func TestClip(t *testing.T) {
	cases := []struct {
		name   string
		window Window
		in     []transcript.Segment
		want   []transcript.Segment
	}{
		{
			name:   "open window keeps everything",
			window: Window{Start: 5},
			in: []transcript.Segment{
				{Start: 6, End: 9, Text: "a b", Words: []transcript.Word{w("a", 6, 7), w("b", 8, 9)}},
			},
			want: []transcript.Segment{
				{Start: 6, End: 9, Text: "a b", Words: []transcript.Word{w("a", 6, 7), w("b", 8, 9)}},
			},
		},
		{
			name:   "word starting at the window end is dropped and the segment rebuilt",
			window: Window{Start: 5, End: 8},
			in: []transcript.Segment{
				{Start: 6, End: 9, Text: "a, b.", Words: []transcript.Word{w("a,", 6, 7), w("b.", 8, 9)}},
			},
			want: []transcript.Segment{
				{Start: 6, End: 7, Text: "a,", Words: []transcript.Word{w("a,", 6, 7)}},
			},
		},
		{
			name:   "word running past the window end is cut back",
			window: Window{Start: 5, End: 8},
			in: []transcript.Segment{
				{Start: 6, End: 8.4, Text: "a", Words: []transcript.Word{w("a", 7.5, 8.4)}},
			},
			want: []transcript.Segment{
				{Start: 7.5, End: 8, Text: "a", Words: []transcript.Word{w("a", 7.5, 8)}},
			},
		},
		{
			name:   "segment left with no words is dropped",
			window: Window{Start: 0, End: 8},
			in: []transcript.Segment{
				{Start: 1, End: 2, Text: "a", Words: []transcript.Word{w("a", 1, 2)}},
				{Start: 8.2, End: 9, Text: "b", Words: []transcript.Word{w("b", 8.2, 9)}},
			},
			want: []transcript.Segment{
				{Start: 1, End: 2, Text: "a", Words: []transcript.Word{w("a", 1, 2)}},
			},
		},
		{
			name:   "wordless segment is clamped or dropped",
			window: Window{Start: 0, End: 8},
			in: []transcript.Segment{
				{Start: 7, End: 9, Text: "[music]"},
				{Start: 8, End: 9, Text: "[music]"},
			},
			want: []transcript.Segment{
				{Start: 7, End: 8, Text: "[music]"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := clip(tc.in, tc.window)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("clip:\n got %+v\nwant %+v", got, tc.want)
			}
		})
	}
}
