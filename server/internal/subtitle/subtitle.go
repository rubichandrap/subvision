package subtitle

import (
	"fmt"
	"io"
	"time"

	"github.com/rubichandrap/subvision/server/internal/transcriber"
)

type Segment struct {
	transcriber.Segment
}

func SRTFromSegments(w io.Writer, segments []Segment) error {
	for i, seg := range segments {
		fmt.Fprintf(w, "%d\n", i+1)
		fmt.Fprintf(w, "%s --> %s\n", formatTimestamp(seg.Start), formatTimestamp(seg.End))
		fmt.Fprintf(w, "%s\n\n", seg.Text)
	}
	return nil
}

func formatTimestamp(seconds float64) string {
	t := time.Duration(seconds * float64(time.Second))
	h := t / time.Hour
	t -= h * time.Hour
	m := t / time.Minute
	t -= m * time.Minute
	s := t / time.Second
	t -= s * time.Second
	ms := t / time.Millisecond

	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}
