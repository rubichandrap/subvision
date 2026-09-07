package transcriber

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestValidateSegmentTimingValidSequences(t *testing.T) {
	cases := []struct {
		name     string
		segments []Segment
	}{
		{
			name:     "empty slice",
			segments: []Segment{},
		},
		{
			name: "single valid segment",
			segments: []Segment{
				{Start: 0.0, End: 2.5, Text: "hello"},
			},
		},
		{
			name: "multiple strictly chronological segments",
			segments: []Segment{
				{Start: 0.0, End: 1.5, Text: "first"},
				{Start: 1.5, End: 3.0, Text: "second touching"},
				{Start: 4.0, End: 5.5, Text: "third with gap"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateSegmentTiming(tc.segments); err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateSegmentTimingRejections(t *testing.T) {
	cases := []struct {
		name          string
		segments      []Segment
		expectedIndex int
		reasonSubstr  string
	}{
		{
			name: "NaN start",
			segments: []Segment{
				{Start: math.NaN(), End: 2.0},
			},
			expectedIndex: 0,
			reasonSubstr:  "finite",
		},
		{
			name: "Inf start",
			segments: []Segment{
				{Start: math.Inf(1), End: 2.0},
			},
			expectedIndex: 0,
			reasonSubstr:  "finite",
		},
		{
			name: "NaN end",
			segments: []Segment{
				{Start: 1.0, End: math.NaN()},
			},
			expectedIndex: 0,
			reasonSubstr:  "finite",
		},
		{
			name: "Inf end",
			segments: []Segment{
				{Start: 1.0, End: math.Inf(-1)},
			},
			expectedIndex: 0,
			reasonSubstr:  "finite",
		},
		{
			name: "negative start",
			segments: []Segment{
				{Start: -0.1, End: 2.0},
			},
			expectedIndex: 0,
			reasonSubstr:  "negative",
		},
		{
			name: "end equal to start",
			segments: []Segment{
				{Start: 1.0, End: 1.0},
			},
			expectedIndex: 0,
			reasonSubstr:  "end must be after its start",
		},
		{
			name: "end before start",
			segments: []Segment{
				{Start: 2.0, End: 1.0},
			},
			expectedIndex: 0,
			reasonSubstr:  "end must be after its start",
		},
		{
			name: "second segment starts before first ends (overlap)",
			segments: []Segment{
				{Start: 0.0, End: 2.0},
				{Start: 1.9, End: 3.0},
			},
			expectedIndex: 1,
			reasonSubstr:  "must not start before segment 1 ends",
		},
		{
			name: "second segment out of order",
			segments: []Segment{
				{Start: 3.0, End: 5.0},
				{Start: 1.0, End: 2.0},
			},
			expectedIndex: 1,
			reasonSubstr:  "must not start before segment 1 ends",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSegmentTiming(tc.segments)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			var valErr *TimingValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("expected error to be *TimingValidationError, got %T (%v)", err, err)
			}

			if valErr.Index != tc.expectedIndex {
				t.Errorf("expected segment index %d, got %d", tc.expectedIndex, valErr.Index)
			}

			if tc.reasonSubstr != "" && !contains(valErr.Error(), tc.reasonSubstr) {
				t.Errorf("expected error message to contain %q, got %q", tc.reasonSubstr, valErr.Error())
			}
		})
	}
}

func TestRescaleWordsProportional(t *testing.T) {
	// Original window: 1.0 to 3.0 (span 2.0).
	// Word 1: 1.2 to 1.6 (relative 0.2/2.0 = 10% to 0.6/2.0 = 30%)
	// Word 2: 2.0 to 2.8 (relative 1.0/2.0 = 50% to 1.8/2.0 = 90%)
	orig := Segment{
		Start: 1.0,
		End:   3.0,
		Words: []Word{
			{Text: "hello", Start: 1.2, End: 1.6},
			{Text: "world", Start: 2.0, End: 2.8},
		},
	}

	// Edited window: 2.0 to 6.0 (span 4.0).
	// Expected word 1: 2.0 + 0.1 * 4.0 = 2.4 to 2.0 + 0.3 * 4.0 = 3.2
	// Expected word 2: 2.0 + 0.5 * 4.0 = 4.0 to 2.0 + 0.9 * 4.0 = 5.6
	edited := Segment{
		Start: 2.0,
		End:   6.0,
	}

	rescaled := RescaleWords(orig, edited)
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
	orig := Segment{
		Start: 10.0,
		End:   20.0,
		Words: []Word{
			{Text: "first", Start: 11.0, End: 13.0}, // 10% to 30%
			{Text: "second", Start: 15.0, End: 19.0}, // 50% to 90%
		},
	}

	// Shifted by +5s with same duration (10s): 15.0 to 25.0
	shifted := Segment{Start: 15.0, End: 25.0}
	rescaledShift := RescaleWords(orig, shifted)
	if len(rescaledShift) != 2 {
		t.Fatalf("expected 2 words, got %d", len(rescaledShift))
	}
	assertApprox(t, rescaledShift[0].Start, 16.0, "shifted word 0 start")
	assertApprox(t, rescaledShift[0].End, 18.0, "shifted word 0 end")
	assertApprox(t, rescaledShift[1].Start, 20.0, "shifted word 1 start")
	assertApprox(t, rescaledShift[1].End, 24.0, "shifted word 1 end")

	// Shrunk to 5s: 10.0 to 15.0
	shrunk := Segment{Start: 10.0, End: 15.0}
	rescaledShrunk := RescaleWords(orig, shrunk)
	if len(rescaledShrunk) != 2 {
		t.Fatalf("expected 2 words, got %d", len(rescaledShrunk))
	}
	assertApprox(t, rescaledShrunk[0].Start, 10.5, "shrunk word 0 start")
	assertApprox(t, rescaledShrunk[0].End, 11.5, "shrunk word 0 end")
	assertApprox(t, rescaledShrunk[1].Start, 12.5, "shrunk word 1 start")
	assertApprox(t, rescaledShrunk[1].End, 14.5, "shrunk word 1 end")
}


func TestRescaleWordsUnchangedBoundariesRetainTimestamps(t *testing.T) {
	orig := Segment{
		Start: 1.5,
		End:   3.5,
		Words: []Word{
			{Text: "stay", Start: 1.8, End: 3.2},
		},
	}

	edited := Segment{
		Start: 1.5,
		End:   3.5,
		Text:  "edited text only",
	}

	rescaled := RescaleWords(orig, edited)
	if len(rescaled) != 1 {
		t.Fatalf("expected 1 word, got %d", len(rescaled))
	}

	if rescaled[0].Start != 1.8 || rescaled[0].End != 3.2 {
		t.Errorf("expected unchanged timestamps (1.8, 3.2), got (%v, %v)", rescaled[0].Start, rescaled[0].End)
	}
}

func TestRescaleWordsDegenerateOrEmpty(t *testing.T) {
	// Empty words
	origEmpty := Segment{Start: 1.0, End: 2.0, Words: nil}
	editedEmpty := Segment{Start: 2.0, End: 4.0}
	if res := RescaleWords(origEmpty, editedEmpty); len(res) != 0 {
		t.Errorf("expected empty words, got %+v", res)
	}

	// Degenerate original span (end <= start)
	origDegenerate := Segment{
		Start: 2.0,
		End:   2.0,
		Words: []Word{{Text: "fixed", Start: 2.0, End: 2.0}},
	}
	edited := Segment{Start: 3.0, End: 5.0}
	res := RescaleWords(origDegenerate, edited)
	if len(res) != 1 || res[0].Start != 2.0 || res[0].End != 2.0 {
		t.Errorf("expected degenerate words to be retained without division by zero, got %+v", res)
	}
}

func assertApprox(t *testing.T, got, want float64, name string) {
	t.Helper()
	diff := got - want
	if diff > 1e-9 || diff < -1e-9 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
