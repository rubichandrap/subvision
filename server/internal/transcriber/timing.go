package transcriber

import (
	"fmt"
	"math"
)

// TimingValidationError records a segment timing validation failure.
type TimingValidationError struct {
	Index  int    // 0-based index of the failing segment
	Reason string // description of why the timing is invalid
}

func (e *TimingValidationError) Error() string {
	return fmt.Sprintf("segment %d %s", e.Index+1, e.Reason)
}

// ValidationError is an alias for TimingValidationError for domain error matching.
type ValidationError = TimingValidationError

// ValidateSegmentTiming rejects edited segments the render cannot use:
// non-finite times, negative starts, ends at or before their start, and
// segments that start before the previous one ends (overlap or out of order).
func ValidateSegmentTiming(segments []Segment) error {
	for i, seg := range segments {
		if !finiteTime(seg.Start) || !finiteTime(seg.End) {
			return &TimingValidationError{
				Index:  i,
				Reason: "must carry finite start and end times",
			}
		}
		if seg.Start < 0 {
			return &TimingValidationError{
				Index:  i,
				Reason: "start must not be negative",
			}
		}
		if seg.End <= seg.Start {
			return &TimingValidationError{
				Index:  i,
				Reason: "end must be after its start",
			}
		}
		if i > 0 && seg.Start < segments[i-1].End {
			return &TimingValidationError{
				Index:  i,
				Reason: fmt.Sprintf("must not start before segment %d ends", i),
			}
		}
	}
	return nil
}

// RescaleWords keeps a segment's whisper-original word timings at their
// relative offsets, scaled into the edited window. Word timings are never
// re-derived from scratch; unchanged boundaries or degenerate original
// windows leave words untouched.
func RescaleWords(original, edited Segment) []Word {
	if len(original.Words) == 0 {
		return original.Words
	}
	if original.Start == edited.Start && original.End == edited.End {
		return original.Words
	}
	span := original.End - original.Start
	if span <= 0 {
		return original.Words
	}
	targetSpan := edited.End - edited.Start
	words := make([]Word, len(original.Words))
	for i, w := range original.Words {
		words[i] = Word{
			Text:  w.Text,
			Start: edited.Start + (w.Start-original.Start)/span*targetSpan,
			End:   edited.Start + (w.End-original.Start)/span*targetSpan,
		}
	}
	return words
}

func finiteTime(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
