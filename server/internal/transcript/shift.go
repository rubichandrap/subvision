package transcript

// Shift translates the start and end times of every Transcription Segment
// and all contained Timed Words by offsetSeconds, returning a fresh slice
// without mutating the input structures.
func Shift(segments []Segment, offsetSeconds float64) []Segment {
	if segments == nil {
		return nil
	}
	shifted := make([]Segment, len(segments))
	for i, s := range segments {
		shifted[i] = Segment{
			Start: s.Start + offsetSeconds,
			End:   s.End + offsetSeconds,
			Text:  s.Text,
		}
		if s.Words != nil {
			shifted[i].Words = make([]Word, len(s.Words))
			for j, w := range s.Words {
				shifted[i].Words[j] = Word{
					Text:  w.Text,
					Start: w.Start + offsetSeconds,
					End:   w.End + offsetSeconds,
				}
			}
		}
	}
	return shifted
}
