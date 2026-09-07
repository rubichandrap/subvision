package transcript_test

import (
	"reflect"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/transcript"
)

func testWord(text string, start, end float64) transcript.Word {
	return transcript.Word{Text: text, Start: start, End: end}
}

func TestSplit(t *testing.T) {
	backToBack := func(count int, length float64) []transcript.Word {
		words := make([]transcript.Word, count)
		for i := range words {
			start := float64(i) * length
			words[i] = testWord("w", start, start+length)
		}
		return words
	}
	joinTexts := func(words []transcript.Word) string {
		text := ""
		for i, w := range words {
			if i > 0 {
				text += " "
			}
			text += w.Text
		}
		return text
	}

	cases := []struct {
		name string
		in   []transcript.Segment
		want []transcript.Segment
	}{
		{
			name: "splits on a word pause >= 0.4s",
			in: []transcript.Segment{{
				Start: 10, End: 14, Text: "one two three four",
				Words: []transcript.Word{
					testWord("one", 10, 10.5),
					testWord("two", 10.6, 11),
					testWord("three", 11.4, 11.9), // gap = 11.4 - 11.0 = 0.4s
					testWord("four", 12.0, 12.5),
				},
			}},
			want: []transcript.Segment{
				{Start: 10, End: 11, Text: "one two", Words: []transcript.Word{
					testWord("one", 10, 10.5),
					testWord("two", 10.6, 11),
				}},
				{Start: 11.4, End: 12.5, Text: "three four", Words: []transcript.Word{
					testWord("three", 11.4, 11.9),
					testWord("four", 12.0, 12.5),
				}},
			},
		},
		{
			name: "does not split on pause < 0.4s",
			in: []transcript.Segment{{
				Start: 10, End: 12, Text: "one two",
				Words: []transcript.Word{
					testWord("one", 10.0, 10.5),
					testWord("two", 10.89, 11.5), // gap = 0.39s
				},
			}},
			want: []transcript.Segment{{
				Start: 10, End: 12, Text: "one two",
				Words: []transcript.Word{
					testWord("one", 10.0, 10.5),
					testWord("two", 10.89, 11.5),
				},
			}},
		},
		{
			name: "forces a cut at twelve words",
			in: []transcript.Segment{{
				Start: 0, End: 6.5, Text: "thirteen back to back words",
				Words: backToBack(13, 0.5),
			}},
			want: func() []transcript.Segment {
				words := backToBack(13, 0.5)
				return []transcript.Segment{
					{Start: 0, End: 6, Text: joinTexts(words[:12]), Words: words[:12]},
					{Start: 6, End: 6.5, Text: joinTexts(words[12:]), Words: words[12:]},
				}
			}(),
		},
		{
			name: "forces a cut at eight seconds",
			in: []transcript.Segment{{
				Start: 0, End: 12, Text: "twelve long words",
				Words: backToBack(12, 1),
			}},
			want: func() []transcript.Segment {
				words := backToBack(12, 1)
				return []transcript.Segment{
					{Start: 0, End: 7, Text: joinTexts(words[:7]), Words: words[:7]},
					{Start: 7, End: 12, Text: joinTexts(words[7:]), Words: words[7:]},
				}
			}(),
		},
		{
			name: "leaves a short segment untouched",
			in: []transcript.Segment{{
				Start: 5, End: 7, Text: "hi there",
				Words: []transcript.Word{
					testWord("hi", 5, 5.5),
					testWord("there", 5.6, 6.5),
				},
			}},
			want: []transcript.Segment{{
				Start: 5, End: 7, Text: "hi there",
				Words: []transcript.Word{
					testWord("hi", 5, 5.5),
					testWord("there", 5.6, 6.5),
				},
			}},
		},
		{
			name: "passes a segment without words through",
			in:   []transcript.Segment{{Start: 1, End: 2, Text: "[music]"}},
			want: []transcript.Segment{{Start: 1, End: 2, Text: "[music]"}},
		},
		{
			name: "keeps every word timestamp verbatim across the split",
			in: []transcript.Segment{{
				Start: 10, End: 20, Text: "alpha beta gamma delta",
				Words: []transcript.Word{
					testWord("alpha", 10.1, 10.9),
					testWord("beta", 11.2, 12.7),
					testWord("gamma", 14.3, 15.8),
					testWord("delta", 16.1, 19.4),
				},
			}},
			want: []transcript.Segment{
				{Start: 10.1, End: 12.7, Text: "alpha beta", Words: []transcript.Word{
					testWord("alpha", 10.1, 10.9),
					testWord("beta", 11.2, 12.7),
				}},
				{Start: 14.3, End: 19.4, Text: "gamma delta", Words: []transcript.Word{
					testWord("gamma", 14.3, 15.8),
					testWord("delta", 16.1, 19.4),
				}},
			},
		},
		{
			name: "nil in yields nil out",
			in:   nil,
			want: nil,
		},
		{
			name: "empty in yields empty out",
			in:   []transcript.Segment{},
			want: []transcript.Segment{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := transcript.Split(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Split() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
