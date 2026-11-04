package transcript

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// Timestamp comparisons tolerate float noise and (more importantly) the
// millisecond rounding the editor applies before sending values back. Anything
// within syncTimeEpsilon is treated as "unchanged" so a no-op save takes the
// clone path instead of rewriting every word's timing.
// Related: transcript.ValidateTiming uses 1e-6 and the client uses 1e-6; the
// ASR word-boundary validation uses 1e-3.
const syncTimeEpsilon = 1e-3

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= syncTimeEpsilon
}

// SyncTextAndWords keeps a segment's display text and Timed Words aligned.
// The stored original segment is the source of truth for timings; a text
// edit with the same token count updates the word texts in place, while a
// token-count-changing edit is rejected so karaoke/pop cannot silently
// desync.
func SyncTextAndWords(original, edited Segment) (Segment, error) {
	if len(original.Words) == 0 {
		if len(edited.Words) == 0 {
			return edited, nil
		}
		return syncExistingWords(edited)
	}

	if !nearlyEqual(edited.Start, original.Start) || !nearlyEqual(edited.End, original.End) {
		edited.Words = RescaleWords(original, edited)
	} else {
		edited.Words = append([]Word(nil), original.Words...)
	}

	return syncExistingWords(edited)
}

// SyncManualSegment prepares a user-created segment whose text is the source
// of truth. If the caller already carries words, it validates and repairs
// them like an ordinary save; otherwise it distributes the cue window across
// token-length weights so the manual cue is renderable by karaoke/pop. Empty
// manual text is rejected.
func SyncManualSegment(edited Segment) (Segment, error) {
	if len(edited.Words) > 0 {
		return syncExistingWords(edited)
	}

	tokens := strings.Fields(edited.Text)
	if len(tokens) == 0 {
		return Segment{}, fmt.Errorf("segment text cannot be empty")
	}

	duration := edited.End - edited.Start
	weights := make([]int, len(tokens))
	totalWeight := 0
	for i, token := range tokens {
		weight := utf8.RuneCountInString(token) + 1
		weights[i] = weight
		totalWeight += weight
	}

	words := make([]Word, 0, len(tokens))
	start := edited.Start
	for i, token := range tokens {
		end := edited.End
		if i < len(tokens)-1 {
			fraction := float64(weights[i]) / float64(totalWeight)
			end = start + duration*fraction
		}
		words = append(words, Word{Text: token, Start: start, End: end})
		start = end
	}

	return Segment{
		Start: edited.Start,
		End:   edited.End,
		Text:  strings.Join(tokens, " "),
		Words: words,
	}, nil
}

func syncExistingWords(edited Segment) (Segment, error) {
	if len(edited.Words) == 0 {
		return edited, nil
	}

	tokens := strings.Fields(edited.Text)
	if len(tokens) != len(edited.Words) {
		return Segment{}, fmt.Errorf(
			"segment text has %d words, want %d timed words",
			len(tokens),
			len(edited.Words),
		)
	}

	words := append([]Word(nil), edited.Words...)
	for i := range words {
		words[i].Text = tokens[i]
	}

	return Segment{
		Start: edited.Start,
		End:   edited.End,
		Text:  strings.Join(tokens, " "),
		Words: words,
	}, nil
}
