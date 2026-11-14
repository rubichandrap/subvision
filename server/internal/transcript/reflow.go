package transcript

import "strings"

// ReflowOptions tunes server-side subtitle reflow. WordsPerPage is the hard
// maximum number of Timed Words one output cue may carry; it mirrors the
// existing Edit Spec `captions.wordsPerPage`. MaxWordGapSeconds, when
// positive, closes a cue at any silence between two Timed Words longer than
// it, so a Caption Page never shows a word long before it is spoken; zero
// disables the gap rule.
type ReflowOptions struct {
	WordsPerPage      int
	MaxCharsPerLine   int
	MaxLines          int
	MinCueSeconds     float64
	MaxWordGapSeconds float64
}

// defaultMaxWordGapSeconds is longer than a pause between spoken phrases but
// shorter than the instrumental breaks between sung lines, and covers the
// stretches where ASR recognised nothing.
const defaultMaxWordGapSeconds = 1.0

// DefaultReflowOptions returns the subtitle-layout defaults used by the
// Parakeet pipeline. `wordsPerPage` mirrors the user's existing caption setting.
// Short cues hold into available silence so they stay readable, and long
// silences always start a new cue.
func DefaultReflowOptions(wordsPerPage int) ReflowOptions {
	return ReflowOptions{
		WordsPerPage:      wordsPerPage,
		MaxCharsPerLine:   42,
		MaxLines:          2,
		MinCueSeconds:     0.8,
		MaxWordGapSeconds: defaultMaxWordGapSeconds,
	}
}

// Reflow splits word-bearing Transcription Segments into cue-sized Segments.
//
// It is pure: Timed Word timestamps pass through unchanged, every source word
// appears exactly once, in order, and Segment text is rebuilt from the words.
// Wordless Segments (for example a music marker) pass through untouched.
func Reflow(segments []Segment, options ReflowOptions) []Segment {
	if options.WordsPerPage <= 0 {
		return segments
	}

	maxCharsPerCue := 0
	if options.MaxCharsPerLine > 0 && options.MaxLines > 0 {
		maxCharsPerCue = options.MaxCharsPerLine * options.MaxLines
	}
	out := make([]Segment, 0, len(segments))
	for _, segment := range segments {
		if len(segment.Words) == 0 {
			out = append(out, segment)
			continue
		}
		out = append(out, reflowSegment(segment, options, maxCharsPerCue)...)
	}
	return out
}

func reflowSegment(segment Segment, options ReflowOptions, maxCharsPerCue int) []Segment {
	cues := make([]Segment, 0, max(1, len(segment.Words)/max(1, options.WordsPerPage)))
	current := make([]Word, 0, options.WordsPerPage)
	currentChars := 0

	for _, word := range segment.Words {
		if len(current) > 0 {
			// Join with one space when accounting for the rendered line width.
			charsWithCandidate := currentChars + 1 + len(word.Text)
			tooManyWords := len(current) >= options.WordsPerPage
			tooWide := maxCharsPerCue > 0 && charsWithCandidate > maxCharsPerCue
			afterSilence := options.MaxWordGapSeconds > 0 &&
				word.Start-current[len(current)-1].End > options.MaxWordGapSeconds
			if tooManyWords || tooWide || afterSilence {
				cues = append(cues, segmentFromWords(current))
				current = make([]Word, 0, options.WordsPerPage)
				currentChars = 0
			}
		}

		if len(current) > 0 {
			currentChars++
		}
		current = append(current, word)
		currentChars += len(word.Text)

		// A sentence-ending punctuation mark is a natural cue boundary: close
		// the current cue so the next phrase starts fresh.
		if endsSentencePunctuation(word.Text) {
			cues = append(cues, segmentFromWords(current))
			current = make([]Word, 0, options.WordsPerPage)
			currentChars = 0
		}
	}

	if len(current) > 0 {
		cues = append(cues, segmentFromWords(current))
	}

	return holdCueWindows(cues, segment.End, options.MinCueSeconds)
}

func endsSentencePunctuation(text string) bool {
	return strings.HasSuffix(text, ".") ||
		strings.HasSuffix(text, "!") ||
		strings.HasSuffix(text, "?")
}

// holdCueWindows extends short cue windows into available silence so viewers
// can read them, without overlapping the next cue or leaving the original
// Segment's window.
func holdCueWindows(cues []Segment, segmentEnd, minCueSeconds float64) []Segment {
	if minCueSeconds <= 0 {
		return cues
	}

	for i := range cues {
		cue := &cues[i]
		nextStart := segmentEnd
		if i+1 < len(cues) {
			nextStart = cues[i+1].Start
		}
		if nextStart <= cue.Start {
			continue
		}

		minEnd := cue.Start + minCueSeconds
		if minEnd > nextStart {
			minEnd = nextStart
		}
		if minEnd > cue.End {
			cue.End = minEnd
		}
	}
	return cues
}
