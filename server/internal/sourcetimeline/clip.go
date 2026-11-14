package sourcetimeline

import (
	"strings"

	"github.com/rubichandrap/subvision/server/internal/transcript"
)

// clip keeps only what lies inside the window, in source time. A Timed Word
// starting at or after the window end is dropped; one running past it is cut
// back to the window end. Word-bearing segments are rebuilt from their
// surviving words and dropped when none survive; wordless segments are
// clamped to the window or dropped when outside it. An open window
// (End <= 0) only bounds the start.
func clip(segments []transcript.Segment, window Window) []transcript.Segment {
	if segments == nil {
		return nil
	}
	if window.End <= 0 {
		return segments
	}
	out := make([]transcript.Segment, 0, len(segments))
	for _, seg := range segments {
		if len(seg.Words) == 0 {
			if seg.End <= window.Start || seg.Start >= window.End {
				continue
			}
			seg.Start = max(seg.Start, window.Start)
			seg.End = min(seg.End, window.End)
			out = append(out, seg)
			continue
		}

		kept := make([]transcript.Word, 0, len(seg.Words))
		for _, word := range seg.Words {
			if word.Start >= window.End || word.End <= window.Start {
				continue
			}
			word.Start = max(word.Start, window.Start)
			word.End = min(word.End, window.End)
			kept = append(kept, word)
		}
		if len(kept) == 0 {
			continue
		}
		if len(kept) == len(seg.Words) && seg.End <= window.End && seg.Start >= window.Start {
			out = append(out, seg)
			continue
		}
		texts := make([]string, len(kept))
		for i, word := range kept {
			texts[i] = word.Text
		}
		out = append(out, transcript.Segment{
			Start: kept[0].Start,
			End:   kept[len(kept)-1].End,
			Text:  strings.Join(texts, " "),
			Words: kept,
		})
	}
	return out
}
