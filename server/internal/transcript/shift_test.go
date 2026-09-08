package transcript_test

import (
	"testing"

	"github.com/rubichandrap/subvision/server/internal/transcript"
)

func TestShiftTranslatesSegmentAndWordBoundaries(t *testing.T) {
	orig := []transcript.Segment{
		{
			Start: 1.0,
			End:   3.5,
			Text:  "hello world",
			Words: []transcript.Word{
				{Text: "hello", Start: 1.1, End: 1.8},
				{Text: "world", Start: 2.0, End: 3.2},
			},
		},
		{
			Start: 4.0,
			End:   6.2,
			Text:  "next sentence",
			Words: []transcript.Word{
				{Text: "next", Start: 4.1, End: 4.8},
				{Text: "sentence", Start: 5.0, End: 6.1},
			},
		},
	}

	offset := 10.5
	shifted := transcript.Shift(orig, offset)

	if len(shifted) != len(orig) {
		t.Fatalf("expected %d segments, got %d", len(orig), len(shifted))
	}

	// Segment 0 checks
	assertApprox(t, shifted[0].Start, 11.5, "segment 0 start")
	assertApprox(t, shifted[0].End, 14.0, "segment 0 end")
	if shifted[0].Text != "hello world" {
		t.Errorf("expected segment 0 text preserved, got %q", shifted[0].Text)
	}
	if len(shifted[0].Words) != 2 {
		t.Fatalf("expected 2 words in segment 0, got %d", len(shifted[0].Words))
	}
	assertApprox(t, shifted[0].Words[0].Start, 11.6, "word 0 start")
	assertApprox(t, shifted[0].Words[0].End, 12.3, "word 0 end")
	assertApprox(t, shifted[0].Words[1].Start, 12.5, "word 1 start")
	assertApprox(t, shifted[0].Words[1].End, 13.7, "word 1 end")

	// Segment 1 checks
	assertApprox(t, shifted[1].Start, 14.5, "segment 1 start")
	assertApprox(t, shifted[1].End, 16.7, "segment 1 end")
	if shifted[1].Text != "next sentence" {
		t.Errorf("expected segment 1 text preserved, got %q", shifted[1].Text)
	}
	if len(shifted[1].Words) != 2 {
		t.Fatalf("expected 2 words in segment 1, got %d", len(shifted[1].Words))
	}
	assertApprox(t, shifted[1].Words[0].Start, 14.6, "word 1.0 start")
	assertApprox(t, shifted[1].Words[0].End, 15.3, "word 1.0 end")
	assertApprox(t, shifted[1].Words[1].Start, 15.5, "word 1.1 start")
	assertApprox(t, shifted[1].Words[1].End, 16.6, "word 1.1 end")
}

func TestShiftPreservesRelativeIntervalsAndOrdering(t *testing.T) {
	orig := []transcript.Segment{
		{
			Start: 0.25,
			End:   2.75,
			Text:  "a b c",
			Words: []transcript.Word{
				{Text: "a", Start: 0.30, End: 0.80},
				{Text: "b", Start: 1.00, End: 1.50},
				{Text: "c", Start: 1.80, End: 2.60},
			},
		},
	}

	offset := 15.125
	shifted := transcript.Shift(orig, offset)

	origSpan := orig[0].End - orig[0].Start
	shiftedSpan := shifted[0].End - shifted[0].Start
	assertApprox(t, shiftedSpan, origSpan, "segment span")

	for i := range orig[0].Words {
		origWordSpan := orig[0].Words[i].End - orig[0].Words[i].Start
		shiftedWordSpan := shifted[0].Words[i].End - shifted[0].Words[i].Start
		assertApprox(t, shiftedWordSpan, origWordSpan, "word span")

		if i > 0 {
			origGap := orig[0].Words[i].Start - orig[0].Words[i-1].End
			shiftedGap := shifted[0].Words[i].Start - shifted[0].Words[i-1].End
			assertApprox(t, shiftedGap, origGap, "inter-word gap")

			if shifted[0].Words[i].Start <= shifted[0].Words[i-1].Start {
				t.Errorf("word ordering not preserved: word %d starts at %f <= word %d start %f",
					i, shifted[0].Words[i].Start, i-1, shifted[0].Words[i-1].Start)
			}
		}
	}
}

func TestShiftZeroOffset(t *testing.T) {
	orig := []transcript.Segment{
		{
			Start: 2.5,
			End:   5.0,
			Text:  "testing zero offset",
			Words: []transcript.Word{
				{Text: "testing", Start: 2.6, End: 3.2},
				{Text: "zero", Start: 3.3, End: 4.0},
				{Text: "offset", Start: 4.1, End: 4.9},
			},
		},
	}

	shifted := transcript.Shift(orig, 0.0)

	assertApprox(t, shifted[0].Start, orig[0].Start, "start")
	assertApprox(t, shifted[0].End, orig[0].End, "end")
	if shifted[0].Text != orig[0].Text {
		t.Errorf("text changed: got %q, want %q", shifted[0].Text, orig[0].Text)
	}
	for i := range orig[0].Words {
		assertApprox(t, shifted[0].Words[i].Start, orig[0].Words[i].Start, "word start")
		assertApprox(t, shifted[0].Words[i].End, orig[0].Words[i].End, "word end")
		if shifted[0].Words[i].Text != orig[0].Words[i].Text {
			t.Errorf("word text changed: got %q, want %q", shifted[0].Words[i].Text, orig[0].Words[i].Text)
		}
	}
}

func TestShiftEmptyAndNilInput(t *testing.T) {
	// Nil segments should return nil without panic
	if got := transcript.Shift(nil, 5.0); got != nil {
		t.Errorf("expected nil for nil input, got %+v", got)
	}

	// Empty slice should return empty slice without panic
	empty := []transcript.Segment{}
	gotEmpty := transcript.Shift(empty, 5.0)
	if gotEmpty == nil {
		t.Errorf("expected non-nil empty slice for empty input, got nil")
	}
	if len(gotEmpty) != 0 {
		t.Errorf("expected 0 segments, got %d", len(gotEmpty))
	}
}

func TestShiftSegmentWithoutWords(t *testing.T) {
	orig := []transcript.Segment{
		{
			Start: 1.0,
			End:   2.0,
			Text:  "no words segment",
			Words: nil,
		},
		{
			Start: 3.0,
			End:   4.0,
			Text:  "empty words segment",
			Words: []transcript.Word{},
		},
	}

	shifted := transcript.Shift(orig, 2.5)

	assertApprox(t, shifted[0].Start, 3.5, "segment 0 start")
	assertApprox(t, shifted[0].End, 4.5, "segment 0 end")
	if shifted[0].Words != nil {
		t.Errorf("expected nil words for segment 0, got %+v", shifted[0].Words)
	}

	assertApprox(t, shifted[1].Start, 5.5, "segment 1 start")
	assertApprox(t, shifted[1].End, 6.5, "segment 1 end")
	if shifted[1].Words == nil || len(shifted[1].Words) != 0 {
		t.Errorf("expected empty non-nil words for segment 1, got %+v", shifted[1].Words)
	}
}

func TestShiftDoesNotMutateOriginal(t *testing.T) {
	orig := []transcript.Segment{
		{
			Start: 1.0,
			End:   2.0,
			Text:  "immutable",
			Words: []transcript.Word{
				{Text: "immutable", Start: 1.1, End: 1.9},
			},
		},
	}

	shifted := transcript.Shift(orig, 10.0)

	// Mutate shifted output
	shifted[0].Start = 999.0
	shifted[0].End = 999.0
	shifted[0].Text = "mutated"
	shifted[0].Words[0].Start = 999.0
	shifted[0].Words[0].End = 999.0
	shifted[0].Words[0].Text = "mutated"

	// Original must remain untouched
	assertApprox(t, orig[0].Start, 1.0, "original start")
	assertApprox(t, orig[0].End, 2.0, "original end")
	if orig[0].Text != "immutable" {
		t.Errorf("original segment text was mutated to %q", orig[0].Text)
	}
	assertApprox(t, orig[0].Words[0].Start, 1.1, "original word start")
	assertApprox(t, orig[0].Words[0].End, 1.9, "original word end")
	if orig[0].Words[0].Text != "immutable" {
		t.Errorf("original word text was mutated to %q", orig[0].Words[0].Text)
	}
}

func TestShiftFractionalAndNegativeOffsets(t *testing.T) {
	orig := []transcript.Segment{
		{
			Start: 10.555,
			End:   12.777,
			Text:  "fractional",
			Words: []transcript.Word{
				{Text: "fractional", Start: 10.666, End: 12.555},
			},
		},
	}

	// Positive fractional
	posShifted := transcript.Shift(orig, 0.333)
	assertApprox(t, posShifted[0].Start, 10.888, "pos shift start")
	assertApprox(t, posShifted[0].End, 13.110, "pos shift end")
	assertApprox(t, posShifted[0].Words[0].Start, 10.999, "pos shift word start")
	assertApprox(t, posShifted[0].Words[0].End, 12.888, "pos shift word end")

	// Negative offset (e.g. reverse translation if needed)
	negShifted := transcript.Shift(orig, -5.555)
	assertApprox(t, negShifted[0].Start, 5.0, "neg shift start")
	assertApprox(t, negShifted[0].End, 7.222, "neg shift end")
	assertApprox(t, negShifted[0].Words[0].Start, 5.111, "neg shift word start")
	assertApprox(t, negShifted[0].Words[0].End, 7.0, "neg shift word end")
}
