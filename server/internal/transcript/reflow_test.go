package transcript_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/transcript"
)

func reflowWord(text string, start, end float64) transcript.Word {
	return transcript.Word{Text: text, Start: start, End: end}
}

func wordsFromSegments(segments []transcript.Segment) []transcript.Word {
	words := make([]transcript.Word, 0)
	for _, segment := range segments {
		words = append(words, segment.Words...)
	}
	return words
}

func TestReflowRespectsWordsPerPage(t *testing.T) {
	words := []transcript.Word{
		reflowWord("one", 0, 0.5),
		reflowWord("two", 0.6, 1),
		reflowWord("three", 1.1, 1.5),
		reflowWord("four", 1.6, 2),
		reflowWord("five", 2.1, 2.5),
	}
	segments := []transcript.Segment{{
		Start: words[0].Start,
		End:   words[len(words)-1].End,
		Text:  "one two three four five",
		Words: words,
	}}

	got := transcript.Reflow(segments, transcript.ReflowOptions{WordsPerPage: 2})
	if len(got) != 3 {
		t.Fatalf("Reflow() produced %d cues, want 3", len(got))
	}
	for i, cue := range got {
		if len(cue.Words) > 2 {
			t.Fatalf("cue %d has %d words, want at most 2", i, len(cue.Words))
		}
	}
	wantTexts := []string{"one two", "three four", "five"}
	for i, cue := range got {
		if cue.Text != wantTexts[i] {
			t.Fatalf("cue %d text = %q, want %q", i, cue.Text, wantTexts[i])
		}
	}
}

func TestReflowRespectsCharacterBudget(t *testing.T) {
	words := []transcript.Word{
		reflowWord("extraordinarily", 0, 1),
		reflowWord("long", 1.1, 1.5),
		reflowWord("word", 1.6, 2),
		reflowWord("another", 2.1, 2.5),
	}
	segments := []transcript.Segment{{
		Start: words[0].Start,
		End:   words[len(words)-1].End,
		Text:  "extraordinarily long word another",
		Words: words,
	}}

	got := transcript.Reflow(segments, transcript.ReflowOptions{
		WordsPerPage:    10,
		MaxCharsPerLine: 10,
		MaxLines:        2,
	})
	if len(got) != 2 {
		t.Fatalf("Reflow() produced %d cues, want 2", len(got))
	}
	if got[0].Text != "extraordinarily long" {
		t.Fatalf("first cue = %q, want %q", got[0].Text, "extraordinarily long")
	}
	if got[1].Text != "word another" {
		t.Fatalf("second cue = %q, want %q", got[1].Text, "word another")
	}
}

func TestReflowKeepsWordsInOrder(t *testing.T) {
	words := []transcript.Word{
		reflowWord("one", 0, 0.5),
		reflowWord("two", 0.6, 1),
		reflowWord("three", 1.1, 1.5),
		reflowWord("four", 1.6, 2),
		reflowWord("five", 2.1, 2.5),
	}
	segments := []transcript.Segment{{
		Start: words[0].Start,
		End:   words[len(words)-1].End,
		Text:  "one two three four five",
		Words: words,
	}}

	got := transcript.Reflow(segments, transcript.ReflowOptions{WordsPerPage: 2})
	if gotWords := wordsFromSegments(got); !reflect.DeepEqual(gotWords, words) {
		t.Fatalf("reflowed words = %+v, want %+v", gotWords, words)
	}
}

func TestReflowRebuildsTextFromWords(t *testing.T) {
	words := []transcript.Word{
		reflowWord("hello,", 0, 0.5),
		reflowWord("world.", 0.6, 1),
		reflowWord("another", 1.1, 1.5),
	}
	segments := []transcript.Segment{{
		Start: words[0].Start,
		End:   words[len(words)-1].End,
		Text:  "hello, world. another",
		Words: words,
	}}

	got := transcript.Reflow(segments, transcript.ReflowOptions{WordsPerPage: 2})
	for i, cue := range got {
		texts := make([]string, len(cue.Words))
		for j, word := range cue.Words {
			texts[j] = word.Text
		}
		if want := strings.Join(texts, " "); cue.Text != want {
			t.Fatalf("cue %d text = %q, want %q", i, cue.Text, want)
		}
	}
}

func TestReflowPassesWordlessSegmentsThrough(t *testing.T) {
	segments := []transcript.Segment{{Start: 1, End: 2, Text: "[music]"}}
	got := transcript.Reflow(segments, transcript.ReflowOptions{WordsPerPage: 2})
	if !reflect.DeepEqual(got, segments) {
		t.Fatalf("Reflow() = %+v, want %+v", got, segments)
	}
}

func TestReflowLeavesUnlimitedWordsPerPageUnchanged(t *testing.T) {
	segments := []transcript.Segment{{
		Start: 0,
		End:   2,
		Text:  "one two",
		Words: []transcript.Word{
			reflowWord("one", 0, 0.5),
			reflowWord("two", 0.6, 1),
		},
	}}
	got := transcript.Reflow(segments, transcript.ReflowOptions{WordsPerPage: 0})
	if !reflect.DeepEqual(got, segments) {
		t.Fatalf("Reflow() = %+v, want %+v", got, segments)
	}
}

func TestReflowHoldsShortCuesWithoutOverlap(t *testing.T) {
	words := []transcript.Word{
		reflowWord("one", 0, 0.2),
		reflowWord("two", 0.3, 0.4),
		reflowWord("three", 2.0, 2.2),
		reflowWord("four", 2.3, 2.4),
	}
	segments := []transcript.Segment{{
		Start: words[0].Start,
		End:   words[len(words)-1].End,
		Text:  "one two three four",
		Words: words,
	}}

	got := transcript.Reflow(segments, transcript.ReflowOptions{
		WordsPerPage:  2,
		MinCueSeconds: 1,
	})
	if len(got) != 2 {
		t.Fatalf("Reflow() produced %d cues, want 2", len(got))
	}
	if got[0].Start != 0 || got[0].End != 1 {
		t.Fatalf("first cue window = %v-%v, want 0-1", got[0].Start, got[0].End)
	}
	if got[1].Start != 2.0 || got[1].End != 2.4 {
		t.Fatalf("second cue window = %v-%v, want 2-2.4", got[1].Start, got[1].End)
	}
}

func TestReflowMicroOverlapValidates(t *testing.T) {
	words := []transcript.Word{
		reflowWord("one", 0, 1),
		// Parakeet can report adjacent word boundaries a few microseconds
		// apart; one cue per word would otherwise inherit that overlap.
		reflowWord("two", 0.999996, 2),
	}
	got := transcript.Reflow([]transcript.Segment{{
		Start: words[0].Start,
		End:   words[len(words)-1].End,
		Text:  "one two",
		Words: words,
	}}, transcript.ReflowOptions{WordsPerPage: 1})
	if len(got) != 2 {
		t.Fatalf("Reflow() produced %d cues, want 2", len(got))
	}
	overlap := got[0].End - got[1].Start
	if overlap <= 0 || overlap >= 0.002 {
		t.Fatalf("fixture overlap = %v, want a positive overlap below the validation tolerance", overlap)
	}
	if err := transcript.ValidateTiming(got); err != nil {
		t.Fatalf("ValidateTiming() rejected reflowed micro-overlap: %v", err)
	}
}

func TestReflowOnParakeetFixture(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/parakeet-spike/fixture.json")
	if err != nil {
		t.Skipf("parakeet fixture unavailable: %v", err)
	}

	var payload struct {
		Segments []transcript.Segment `json:"segments"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("failed to parse parakeet fixture: %v", err)
	}

	got := transcript.Reflow(payload.Segments, transcript.ReflowOptions{
		WordsPerPage:    4,
		MaxCharsPerLine: 42,
		MaxLines:        2,
	})
	for i, cue := range got {
		if len(cue.Words) > 4 {
			t.Fatalf("cue %d has %d words, want at most 4", i, len(cue.Words))
		}
		if cue.Start != cue.Words[0].Start {
			t.Fatalf("cue %d start = %v, want %v", i, cue.Start, cue.Words[0].Start)
		}
		if cue.End != cue.Words[len(cue.Words)-1].End {
			t.Fatalf("cue %d end = %v, want %v", i, cue.End, cue.Words[len(cue.Words)-1].End)
		}
	}
}

func TestReflowSplitsCuesOnSilence(t *testing.T) {
	// A sung line, a long instrumental pause, then the next line: the pause
	// must not sit inside one Caption Page, or the later words show on
	// screen long before they are sung.
	words := []transcript.Word{
		reflowWord("let", 21.36, 21.68),
		reflowWord("me", 21.68, 21.84),
		reflowWord("show", 21.84, 22.24),
		reflowWord("you", 22.24, 22.40),
		reflowWord("love", 22.56, 22.88),
		reflowWord("stay", 25.04, 25.20),
		reflowWord("by", 25.36, 25.52),
	}
	segments := []transcript.Segment{{
		Start: words[0].Start,
		End:   words[len(words)-1].End,
		Text:  "let me show you love stay by",
		Words: words,
	}}

	got := transcript.Reflow(segments, transcript.ReflowOptions{
		WordsPerPage:      4,
		MaxWordGapSeconds: 1,
	})
	wantTexts := []string{"let me show you", "love", "stay by"}
	if len(got) != len(wantTexts) {
		t.Fatalf("Reflow() produced %d cues %v, want %v", len(got), cueTexts(got), wantTexts)
	}
	for i, cue := range got {
		if cue.Text != wantTexts[i] {
			t.Fatalf("cue %d text = %q, want %q", i, cue.Text, wantTexts[i])
		}
	}
	if !reflect.DeepEqual(wordsFromSegments(got), words) {
		t.Fatal("Reflow() moved, dropped, or reordered a Timed Word")
	}
}

func TestReflowKeepsCueAcrossGapAtThreshold(t *testing.T) {
	words := []transcript.Word{
		reflowWord("one", 0, 0.5),
		reflowWord("two", 1.5, 2), // exactly 1 s of silence
	}
	got := transcript.Reflow([]transcript.Segment{{
		Start: 0, End: 2, Text: "one two", Words: words,
	}}, transcript.ReflowOptions{WordsPerPage: 4, MaxWordGapSeconds: 1})
	if len(got) != 1 {
		t.Fatalf("Reflow() produced %d cues %v, want 1", len(got), cueTexts(got))
	}
}

func TestReflowWithoutGapLimitKeepsLongPausesTogether(t *testing.T) {
	words := []transcript.Word{
		reflowWord("one", 0, 0.5),
		reflowWord("two", 10, 10.5),
	}
	got := transcript.Reflow([]transcript.Segment{{
		Start: 0, End: 10.5, Text: "one two", Words: words,
	}}, transcript.ReflowOptions{WordsPerPage: 4})
	if len(got) != 1 {
		t.Fatalf("Reflow() produced %d cues %v, want 1 with no gap limit", len(got), cueTexts(got))
	}
}

func TestDefaultReflowNeverHoldsAWordOnScreenThroughSilence(t *testing.T) {
	// Song-shaped input: no punctuation, short words, irregular pauses up to
	// half a minute where recognition found nothing.
	starts := []float64{16.08, 16.40, 16.56, 16.96, 17.44, 21.36, 29.04, 29.36, 35.28, 60.64, 61.0, 79.2, 119.0, 148.0}
	words := make([]transcript.Word, len(starts))
	for i, start := range starts {
		words[i] = reflowWord(fmt.Sprintf("w%d", i), start, start+0.16)
	}
	got := transcript.Reflow([]transcript.Segment{{
		Start: words[0].Start, End: words[len(words)-1].End, Words: words,
	}}, transcript.DefaultReflowOptions(4))

	limit := transcript.DefaultReflowOptions(4).MaxWordGapSeconds
	if limit <= 0 {
		t.Fatal("DefaultReflowOptions must set a positive MaxWordGapSeconds")
	}
	for i, cue := range got {
		for j := 1; j < len(cue.Words); j++ {
			if gap := cue.Words[j].Start - cue.Words[j-1].End; gap > limit {
				t.Fatalf("cue %d %q holds %.1fs of silence before %q", i, cue.Text, gap, cue.Words[j].Text)
			}
		}
	}
	if err := transcript.ValidateTiming(got); err != nil {
		t.Fatalf("ValidateTiming() rejected silence-split cues: %v", err)
	}
}

func cueTexts(cues []transcript.Segment) []string {
	out := make([]string, len(cues))
	for i, cue := range cues {
		out[i] = cue.Text
	}
	return out
}
