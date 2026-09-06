package transcriber

import "strings"

// Split tuning: a new segment starts at a word gap of at least this many
// seconds; the word and duration caps force a cut when speech never pauses.
const (
	splitPauseSeconds = 0.4
	splitMaxWords     = 12
	splitMaxSeconds   = 8.0
)

// SplitSegments cuts long Transcription Segments into shorter ones on natural
// speech pauses: a word starting at least splitPauseSeconds after the previous
// word ends begins a new segment. The word-count and duration caps force a cut
// mid-flow when there is no pause. Word timestamps pass through verbatim and
// each piece's text is rebuilt from its words; segments without words and
// unsplittable ones pass through untouched.
func SplitSegments(segments []Segment) []Segment {
	if segments == nil {
		return nil
	}
	out := make([]Segment, 0, len(segments))
	for _, segment := range segments {
		out = append(out, splitOne(segment)...)
	}
	return out
}

func splitOne(segment Segment) []Segment {
	if len(segment.Words) == 0 {
		return []Segment{segment}
	}
	boundaries := []int{0}
	anchor := segment.Words[0].Start
	for i := 1; i < len(segment.Words); i++ {
		gap := segment.Words[i].Start - segment.Words[i-1].End
		inPiece := i - boundaries[len(boundaries)-1]
		spanWithCandidate := segment.Words[i].End - anchor
		// A cut starts a new piece at word i: caps keep every piece within
		// bounds, a pause keeps pieces on natural speech breaks.
		if inPiece >= splitMaxWords || spanWithCandidate >= splitMaxSeconds || gap >= splitPauseSeconds {
			boundaries = append(boundaries, i)
			anchor = segment.Words[i].Start
		}
	}
	if len(boundaries) == 1 {
		return []Segment{segment}
	}
	boundaries = append(boundaries, len(segment.Words))
	out := make([]Segment, 0, len(boundaries)-1)
	for i := 0; i+1 < len(boundaries); i++ {
		out = append(out, segmentFromWords(segment.Words[boundaries[i]:boundaries[i+1]]))
	}
	return out
}

// segmentFromWords rebuilds a Segment from its words: the text is the word
// texts joined, the window is the first and last word's bounds.
func segmentFromWords(words []Word) Segment {
	texts := make([]string, len(words))
	for i, word := range words {
		texts[i] = word.Text
	}
	return Segment{
		Start: words[0].Start,
		End:   words[len(words)-1].End,
		Text:  strings.Join(texts, " "),
		Words: words,
	}
}
