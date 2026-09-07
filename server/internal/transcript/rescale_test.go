package transcript_test

import (
	"math"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/transcript"
)

func assertApprox(t *testing.T, got, want float64, name string) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("%s = %f, want %f", name, got, want)
	}
}

func TestRescaleWordsProportional(t *testing.T) {
	// Original window: 1.0 to 3.0 (span 2.0).
	// Word 0: 1.2 to 1.6 (relative (1.2-1.0)/2.0 = 0.1 to (1.6-1.0)/2.0 = 0.3)
	// Word 1: 2.0 to 2.8 (relative (2.0-1.0)/2.0 = 0.5 to (2.8-1.0)/2.0 = 0.9)
	orig := transcript.Segment{
		Start: 1.0,
		End:   3.0,
		Text:  "hello world",
		Words: []transcript.Word{
			{Text: "hello", Start: 1.2, End: 1.6},
			{Text: "world", Start: 2.0, End: 2.8},
		},
	}

	// Edited window: 2.0 to 6.0 (span 4.0).
	// Expected word 0: 2.0 + 0.1 * 4.0 = 2.4 to 2.0 + 0.3 * 4.0 = 3.2
	// Expected word 1: 2.0 + 0.5 * 4.0 = 4.0 to 2.0 + 0.9 * 4.0 = 5.6
	edited := transcript.Segment{
		Start: 2.0,
		End:   6.0,
		Text:  "hello world",
	}

	rescaled := transcript.RescaleWords(orig, edited)
	if len(rescaled) != 2 {
		t.Fatalf("expected 2 words, got %d", len(rescaled))
	}

	assertApprox(t, rescaled[0].Start, 2.4, "word 0 start")
	assertApprox(t, rescaled[0].End, 3.2, "word 0 end")
	assertApprox(t, rescaled[1].Start, 4.0, "word 1 start")
	assertApprox(t, rescaled[1].End, 5.6, "word 1 end")
	if rescaled[0].Text != "hello" || rescaled[1].Text != "world" {
		t.Errorf("word texts mutated: %+v", rescaled)
	}
}

func TestRescaleWordsShiftAndShrink(t *testing.T) {
	orig := transcript.Segment{
		Start: 10.0,
		End:   20.0,
		Text:  "one two",
		Words: []transcript.Word{
			{Text: "one", Start: 12.0, End: 15.0},
			{Text: "two", Start: 16.0, End: 18.0},
		},
	}

	// Shifted earlier and duration halved (span 5.0)
	edited := transcript.Segment{
		Start: 5.0,
		End:   10.0,
		Text:  "one two",
	}

	rescaled := transcript.RescaleWords(orig, edited)
	if len(rescaled) != 2 {
		t.Fatalf("expected 2 words, got %d", len(rescaled))
	}

	// Word 0 relative: [0.2, 0.5] -> 5.0 + 0.2*5 = 6.0, 5.0 + 0.5*5 = 7.5
	assertApprox(t, rescaled[0].Start, 6.0, "word 0 start")
	assertApprox(t, rescaled[0].End, 7.5, "word 0 end")
	// Word 1 relative: [0.6, 0.8] -> 5.0 + 0.6*5 = 8.0, 5.0 + 0.8*5 = 9.0
	assertApprox(t, rescaled[1].Start, 8.0, "word 1 start")
	assertApprox(t, rescaled[1].End, 9.0, "word 1 end")
}

func TestRescaleWordsUnchangedBoundariesRetainTimestamps(t *testing.T) {
	orig := transcript.Segment{
		Start: 1.0,
		End:   3.0,
		Text:  "same",
		Words: []transcript.Word{
			{Text: "same", Start: 1.2, End: 2.8},
		},
	}
	edited := transcript.Segment{
		Start: 1.0,
		End:   3.0,
		Text:  "same (edited text)",
	}

	rescaled := transcript.RescaleWords(orig, edited)
	if len(rescaled) != 1 {
		t.Fatalf("expected 1 word, got %d", len(rescaled))
	}
	if rescaled[0].Start != 1.2 || rescaled[0].End != 2.8 {
		t.Errorf("expected untouched timestamps [1.2, 2.8], got [%f, %f]", rescaled[0].Start, rescaled[0].End)
	}
}

func TestRescaleWordsDegenerateOrEmpty(t *testing.T) {
	t.Run("empty words", func(t *testing.T) {
		orig := transcript.Segment{Start: 1.0, End: 2.0}
		edited := transcript.Segment{Start: 3.0, End: 5.0}
		rescaled := transcript.RescaleWords(orig, edited)
		if len(rescaled) != 0 {
			t.Errorf("expected empty words, got %+v", rescaled)
		}
	})

	t.Run("degenerate original span", func(t *testing.T) {
		orig := transcript.Segment{
			Start: 2.0,
			End:   2.0,
			Words: []transcript.Word{{Text: "w", Start: 2.0, End: 2.0}},
		}
		edited := transcript.Segment{Start: 1.0, End: 3.0}
		rescaled := transcript.RescaleWords(orig, edited)
		if len(rescaled) != 1 || rescaled[0].Start != 2.0 {
			t.Errorf("expected original words returned on degenerate span, got %+v", rescaled)
		}
	})
}
