package transcript

import (
	"fmt"
	"math"
)

// ValidationError records a segment timing validation failure.
type ValidationError struct {
	Index  int    // 0-based index of the failing segment
	Reason string // description of why the timing is invalid
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("segment %d %s", e.Index+1, e.Reason)
}
// ValidateTiming rejects segment sequences with timing anomalies:
// non-finite start or end times, negative start times, ends at or before
// their start, and segments that start before the previous one ends.
func ValidateTiming(segments []Segment) error {
	for i, seg := range segments {
		if !finiteTime(seg.Start) || !finiteTime(seg.End) {
			return &ValidationError{
				Index:  i,
				Reason: "must carry finite start and end times",
			}
		}
		if seg.Start < 0 {
			return &ValidationError{
				Index:  i,
				Reason: "start must not be negative",
			}
		}
		if seg.End <= seg.Start {
			return &ValidationError{
				Index:  i,
				Reason: "end must be after its start",
			}
		}
		if i > 0 && seg.Start < segments[i-1].End {
			return &ValidationError{
				Index:  i,
				Reason: fmt.Sprintf("must not start before segment %d ends", i),
			}
		}
	}
	return nil
}

func finiteTime(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
