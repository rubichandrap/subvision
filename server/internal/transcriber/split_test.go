package transcriber

import (
	"reflect"
	"testing"
)

func testWord(text string, start, end float64) Word {
	return Word{Text: text, Start: start, End: end}
}

func TestSplitSegments(t *testing.T) {
	backToBack := func(count int, length float64) []Word {
		words := make([]Word, count)
		for i := range words {
			start := float64(i) * length
			words[i] = testWord("w", start, start+length)
		}
		return words
	}
	joinTexts := func(words []Word) string {
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
		in   []Segment
		want []Segment
	}{
		{
			name: "splits on a word pause",
			in: []Segment{{
				Start: 10, End: 14, Text: "one two three four",
				Words: []Word{
					testWord("one", 10, 10.5),
					testWord("two", 10.6, 11),
					testWord("three", 12, 12.5),
					testWord("four", 12.6, 13),
				},
			}},
			want: []Segment{
				{Start: 10, End: 11, Text: "one two", Words: []Word{
					testWord("one", 10, 10.5),
					testWord("two", 10.6, 11),
				}},
				{Start: 12, End: 13, Text: "three four", Words: []Word{
					testWord("three", 12, 12.5),
					testWord("four", 12.6, 13),
				}},
			},
		},
		{
			name: "forces a cut at twelve words",
			in: []Segment{{
				Start: 0, End: 6.5, Text: "thirteen back to back words",
				Words: backToBack(13, 0.5),
			}},
			want: func() []Segment {
				words := backToBack(13, 0.5)
				return []Segment{
					{Start: 0, End: 6, Text: joinTexts(words[:12]), Words: words[:12]},
					{Start: 6, End: 6.5, Text: joinTexts(words[12:]), Words: words[12:]},
				}
			}(),
		},
		{
			name: "forces a cut at eight seconds",
			in: []Segment{{
				Start: 0, End: 12, Text: "twelve long words",
				Words: backToBack(12, 1),
			}},
			want: func() []Segment {
				words := backToBack(12, 1)
				return []Segment{
					{Start: 0, End: 7, Text: joinTexts(words[:7]), Words: words[:7]},
					{Start: 7, End: 12, Text: joinTexts(words[7:]), Words: words[7:]},
				}
			}(),
		},
		{
			name: "leaves a short segment untouched",
			in: []Segment{{
				Start: 5, End: 7, Text: "hi there",
				Words: []Word{
					testWord("hi", 5, 5.5),
					testWord("there", 5.6, 6.5),
				},
			}},
			want: []Segment{{
				Start: 5, End: 7, Text: "hi there",
				Words: []Word{
					testWord("hi", 5, 5.5),
					testWord("there", 5.6, 6.5),
				},
			}},
		},
		{
			name: "passes a segment without words through",
			in:   []Segment{{Start: 1, End: 2, Text: "[music]"}},
			want: []Segment{{Start: 1, End: 2, Text: "[music]"}},
		},
		{
			name: "keeps every word timestamp verbatim across the split",
			in: []Segment{{
				Start: 10, End: 20, Text: "alpha beta gamma delta",
				Words: []Word{
					testWord("alpha", 10.1, 10.9),
					testWord("beta", 11.2, 12.7),
					testWord("gamma", 14.3, 15.8),
					testWord("delta", 16.1, 19.4),
				},
			}},
			want: []Segment{
				{Start: 10.1, End: 12.7, Text: "alpha beta", Words: []Word{
					testWord("alpha", 10.1, 10.9),
					testWord("beta", 11.2, 12.7),
				}},
				{Start: 14.3, End: 19.4, Text: "gamma delta", Words: []Word{
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SplitSegments(tc.in); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("SplitSegments() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
