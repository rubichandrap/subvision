// Package sourcetimeline owns the Source Timeline: the container timeline
// the rendered Output plays on, where every persisted Transcription Segment
// and Timed Word lives.
//
// Transcribe hides the whole round trip from a video to source-time
// segments: it extracts the trim window's audio so that WAV time 0 is
// exactly the window start on the Source Timeline, sends it to the ASR
// Worker, translates the result back into source time, and clips it to the
// window. See ADR-0011 for why the extraction flags matter.
package sourcetimeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/rubichandrap/subvision/server/internal/transcript"
)

// Window is the trim window in source seconds. End <= 0 means "to the end
// of the video".
type Window struct {
	Start float64
	End   float64
}

// ASR turns a 16 kHz mono 16-bit WAV into Transcription Segments timed from
// the start of that WAV.
type ASR interface {
	Transcribe(audioPath string) ([]transcript.Segment, error)
}

// Timeline transcribes videos onto the Source Timeline.
type Timeline struct {
	asr        ASR
	scratchDir string
}

// New returns a Timeline that sends extracted audio to asr and keeps its
// scratch WAVs in scratchDir (the OS temp dir when empty).
func New(asr ASR, scratchDir string) *Timeline {
	return &Timeline{asr: asr, scratchDir: scratchDir}
}

// Transcribe returns the window's Transcription Segments in source time,
// clipped to the window. It fails when extraction or the ASR Worker fails;
// an empty transcript is a successful result.
func (t *Timeline) Transcribe(ctx context.Context, videoPath string, window Window) ([]transcript.Segment, error) {
	scratch, err := os.CreateTemp(t.scratchDir, "source-audio-*.wav")
	if err != nil {
		return nil, fmt.Errorf("create scratch audio: %w", err)
	}
	audioPath := scratch.Name()
	_ = scratch.Close()
	defer os.Remove(audioPath)

	if err := extract(ctx, videoPath, audioPath, window); err != nil {
		return nil, err
	}

	segments, err := t.asr.Transcribe(audioPath)
	if err != nil {
		return nil, err
	}
	if window.Start > 0 {
		segments = shift(segments, window.Start)
	}
	return clip(segments, window), nil
}

// extract writes the window's audio as 16 kHz mono PCM. Two flags carry the
// Source Timeline into the WAV (ADR-0011):
//
//   - aresample=async=1:first_pts=0 pads a late-starting audio stream and
//     fills timestamp gaps with silence, so WAV time tracks container time
//     instead of sample count;
//   - -t (end-start) bounds the window as a duration; after an input -ss an
//     output -to would act as a duration from the seek point.
func extract(ctx context.Context, videoPath, audioPath string, window Window) error {
	args := []string{"-hide_banner", "-loglevel", "error", "-y"}
	if window.Start > 0 {
		args = append(args, "-ss", seconds(window.Start))
	}
	args = append(args, "-i", videoPath)
	if window.End > 0 {
		if window.End <= window.Start {
			return fmt.Errorf("trim window ends at %ss, before its start %ss", seconds(window.End), seconds(window.Start))
		}
		args = append(args, "-t", seconds(window.End-window.Start))
	}
	args = append(args,
		"-vn",
		"-af", "aresample=async=1:first_pts=0",
		"-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1",
		audioPath,
	)
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extract audio with ffmpeg: %w: %s", err, out)
	}
	return nil
}

func seconds(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}
