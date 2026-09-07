package transcript

// Word represents one Timed Word of a Transcription Segment with its own
// start and end time. Word timings are never guessed from segment duration.
type Word struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Segment represents one timed subtitle unit: start time, end time, text,
// and the Timed Words contained inside it.
type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
	Words []Word  `json:"words"`
}
