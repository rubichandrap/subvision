package sourcetimeline

import "github.com/rubichandrap/subvision/server/internal/transcript"

// shift translates the start and end times of every Transcription Segment
// and all contained Timed Words by offsetSeconds, returning a fresh slice
// without mutating the input structures.
func shift(segments []transcript.Segment, offsetSeconds float64) []transcript.Segment {
	if segments == nil {
		return nil
	}
	shifted := make([]transcript.Segment, len(segments))
	for i, s := range segments {
		shifted[i] = transcript.Segment{
			Start: s.Start + offsetSeconds,
			End:   s.End + offsetSeconds,
			Text:  s.Text,
		}
		if s.Words != nil {
			shifted[i].Words = make([]transcript.Word, len(s.Words))
			for j, w := range s.Words {
				shifted[i].Words[j] = transcript.Word{
					Text:  w.Text,
					Start: w.Start + offsetSeconds,
					End:   w.End + offsetSeconds,
				}
			}
		}
	}
	return shifted
}
