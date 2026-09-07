package transcript

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
