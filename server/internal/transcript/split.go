package transcript

import "strings"

// Split tuning per ADR-0005: a new segment starts at a natural speech pause
// of at least 0.4 seconds; maximum word count (12) and duration (8.0s) caps
// force a cut when speech never pauses.
const (
	SplitPauseSeconds = 0.4
	SplitMaxWords     = 12
	SplitMaxSeconds   = 8.0
)

// Split cuts long Transcription Segments into shorter ones on natural
// speech pauses (>= 0.4s), capping pieces at 12 words and 8.0s duration per ADR-0005.
// Word timestamps pass through verbatim and each piece's text is rebuilt from
// its words; segments without words and unsplittable ones pass through untouched.
func Split(segments []Segment) []Segment {
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
		if inPiece >= SplitMaxWords || spanWithCandidate >= SplitMaxSeconds || gap >= SplitPauseSeconds {
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
// texts joined with spaces, the window is the first and last word's bounds.
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
