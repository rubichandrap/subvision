package transcript_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/transcript"
)

func TestValidateTimingValidSequences(t *testing.T) {
	cases := []struct {
		name     string
		segments []transcript.Segment
	}{
		{
			name:     "empty sequence",
			segments: nil,
		},
		{
			name:     "empty slice",
			segments: []transcript.Segment{},
		},
		{
			name: "single segment",
			segments: []transcript.Segment{
				{Start: 0.0, End: 1.5, Text: "hello"},
			},
		},
		{
			name: "multiple ordered segments with gaps",
			segments: []transcript.Segment{
				{Start: 0.0, End: 1.5, Text: "hello"},
				{Start: 2.0, End: 3.5, Text: "world"},
				{Start: 4.0, End: 5.0, Text: "again"},
			},
		},
		{
			name: "contiguous segments",
			segments: []transcript.Segment{
				{Start: 0.0, End: 1.5, Text: "hello"},
				{Start: 1.5, End: 3.0, Text: "world"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := transcript.ValidateTiming(tc.segments); err != nil {
				t.Fatalf("unexpected error for valid sequence: %v", err)
			}
		})
	}
}

func TestValidateTimingRejections(t *testing.T) {
	cases := []struct {
		name       string
		segments   []transcript.Segment
		wantIndex  int
		wantReason string
	}{
		{
			name: "nan start",
			segments: []transcript.Segment{
				{Start: math.NaN(), End: 1.0, Text: "invalid"},
			},
			wantIndex:  0,
			wantReason: "must carry finite start and end times",
		},
		{
			name: "nan end",
			segments: []transcript.Segment{
				{Start: 0.0, End: math.NaN(), Text: "invalid"},
			},
			wantIndex:  0,
			wantReason: "must carry finite start and end times",
		},
		{
			name: "infinite start",
			segments: []transcript.Segment{
				{Start: math.Inf(1), End: 5.0, Text: "invalid"},
			},
			wantIndex:  0,
			wantReason: "must carry finite start and end times",
		},
		{
			name: "infinite end",
			segments: []transcript.Segment{
				{Start: 0.0, End: math.Inf(-1), Text: "invalid"},
			},
			wantIndex:  0,
			wantReason: "must carry finite start and end times",
		},
		{
			name: "negative start",
			segments: []transcript.Segment{
				{Start: -0.1, End: 1.0, Text: "negative"},
			},
			wantIndex:  0,
			wantReason: "start must not be negative",
		},
		{
			name: "end equal to start",
			segments: []transcript.Segment{
				{Start: 1.0, End: 1.0, Text: "zero duration"},
			},
			wantIndex:  0,
			wantReason: "end must be after its start",
		},
		{
			name: "end before start",
			segments: []transcript.Segment{
				{Start: 2.0, End: 1.0, Text: "reversed"},
			},
			wantIndex:  0,
			wantReason: "end must be after its start",
		},
		{
			name: "second segment starts before first ends",
			segments: []transcript.Segment{
				{Start: 0.0, End: 2.0, Text: "first"},
				{Start: 1.5, End: 3.0, Text: "second"},
			},
			wantIndex:  1,
			wantReason: "must not start before segment 1 ends",
		},
		{
			name: "third segment starts before second ends",
			segments: []transcript.Segment{
				{Start: 0.0, End: 1.0, Text: "first"},
				{Start: 1.0, End: 3.0, Text: "second"},
				{Start: 2.5, End: 4.0, Text: "third"},
			},
			wantIndex:  2,
			wantReason: "must not start before segment 2 ends",
		},
		{
			name: "out of order segments",
			segments: []transcript.Segment{
				{Start: 3.0, End: 4.0, Text: "late"},
				{Start: 1.0, End: 2.0, Text: "early"},
			},
			wantIndex:  1,
			wantReason: "must not start before segment 1 ends",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := transcript.ValidateTiming(tc.segments)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tc.name)
			}

			var valErr *transcript.ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("expected *transcript.ValidationError, got %T: %v", err, err)
			}

			if valErr.Index != tc.wantIndex {
				t.Errorf("Index = %d, want %d", valErr.Index, tc.wantIndex)
			}
			if !strings.Contains(valErr.Reason, tc.wantReason) {
				t.Errorf("Reason = %q, want it to contain %q", valErr.Reason, tc.wantReason)
			}
			if !strings.Contains(valErr.Error(), tc.wantReason) {
				t.Errorf("Error() = %q, want it to contain %q", valErr.Error(), tc.wantReason)
			}
		})
	}
}
