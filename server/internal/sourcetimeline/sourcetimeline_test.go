package sourcetimeline_test

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/sourcetimeline"
	"github.com/rubichandrap/subvision/server/internal/transcript"
)

// These tests cross the module's real interface: a generated video goes
// through real ffmpeg, and a fake ASR Worker "hears" tone bursts in the WAV
// it receives, reporting one Timed Word per burst at the burst's onset. A
// word's source time therefore proves where the extracted audio sits on the
// Source Timeline.

const tolerance = 0.05 // seconds; well under the drift being fixed

func requireFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
}

func ffmpeg(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("ffmpeg", append([]string{"-hide_banner", "-loglevel", "error", "-y"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg %v: %v\n%s", args, err, out)
	}
}

// toneVideo builds a small video whose audio is a 440 Hz tone, audible only
// where audible(t) is true, then optionally offsets the audio stream start.
func toneVideo(t *testing.T, seconds float64, audibleExpr string, filter string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "video.mp4")
	af := fmt.Sprintf("volume=enable='not(%s)':volume=0", audibleExpr)
	if filter != "" {
		af += "," + filter
	}
	ffmpeg(t,
		"-f", "lavfi", "-i", fmt.Sprintf("testsrc=d=%g:r=10:s=64x64", seconds+1),
		"-f", "lavfi", "-i", fmt.Sprintf("sine=f=440:d=%g", seconds),
		"-filter_complex", "[1:a]"+af+"[a]",
		"-map", "0:v", "-map", "[a]", "-c:v", "libx264", "-c:a", "aac", path)
	return path
}

// offsetAudio remuxes path so the audio stream starts offset seconds after
// the video stream, as many downloaded/remuxed music videos do.
func offsetAudio(t *testing.T, path string, offset float64) string {
	t.Helper()
	out := filepath.Join(filepath.Dir(path), "offset.mp4")
	ffmpeg(t, "-i", path, "-itsoffset", fmt.Sprintf("%g", offset), "-i", path,
		"-map", "0:v", "-map", "1:a", "-c", "copy", out)
	return out
}

type burstASR struct {
	duration float64
	err      error
}

func (b *burstASR) Transcribe(audioPath string) ([]transcript.Segment, error) {
	if b.err != nil {
		return nil, b.err
	}
	samples, err := readWAV(audioPath)
	if err != nil {
		return nil, err
	}
	b.duration = float64(len(samples)) / 16000
	var words []transcript.Word
	const window = 160 // 10 ms
	loud := false
	for i := 0; i+window <= len(samples); i += window {
		var energy float64
		for _, s := range samples[i : i+window] {
			energy += float64(s) * float64(s)
		}
		rms := math.Sqrt(energy / window)
		at := float64(i) / 16000
		switch {
		case !loud && rms > 1000:
			loud = true
			words = append(words, transcript.Word{Text: fmt.Sprintf("w%d", len(words)+1), Start: at, End: at})
		case loud && rms < 200:
			loud = false
			words[len(words)-1].End = at
		}
	}
	if loud {
		words[len(words)-1].End = b.duration
	}
	if len(words) == 0 {
		return nil, nil
	}
	segments := make([]transcript.Segment, 0, len(words))
	for _, w := range words {
		segments = append(segments, transcript.Segment{Start: w.Start, End: w.End, Text: w.Text, Words: []transcript.Word{w}})
	}
	return segments, nil
}

func readWAV(path string) ([]int16, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, errors.New("not a WAV file")
	}
	for pos := 12; pos+8 <= len(data); {
		id := string(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		body := pos + 8
		if id == "data" {
			end := min(body+size, len(data))
			out := make([]int16, (end-body)/2)
			for i := range out {
				out[i] = int16(binary.LittleEndian.Uint16(data[body+2*i:]))
			}
			return out, nil
		}
		pos = body + size + size%2
	}
	return nil, errors.New("WAV has no data chunk")
}

func starts(segments []transcript.Segment) []float64 {
	var out []float64
	for _, s := range segments {
		for _, w := range s.Words {
			out = append(out, w.Start)
		}
	}
	return out
}

func assertStarts(t *testing.T, segments []transcript.Segment, want ...float64) {
	t.Helper()
	got := starts(segments)
	if len(got) != len(want) {
		t.Fatalf("got word starts %v, want %v", got, want)
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > tolerance {
			t.Fatalf("word %d starts at %.3fs, want %.3fs (all: %v)", i+1, got[i], want[i], got)
		}
	}
}

func TestAudioStartOffsetIsKept(t *testing.T) {
	requireFFmpeg(t)
	// Tone audible from 1 s of the audio stream, which itself starts 0.75 s
	// into the container: the word is spoken at 1.75 s source time.
	video := offsetAudio(t, toneVideo(t, 4, "gte(t,1)", ""), 0.75)
	asr := &burstASR{}
	segments, err := sourcetimeline.New(asr, t.TempDir()).Transcribe(context.Background(), video, sourcetimeline.Window{})
	if err != nil {
		t.Fatal(err)
	}
	assertStarts(t, segments, 1.75)
}

func TestAudioTimestampGapIsKept(t *testing.T) {
	requireFFmpeg(t)
	// Bursts at 1 s and 4 s of audio; a 1 s pts gap is inserted at 3 s, so
	// the second burst is heard at 5 s source time. Collapsing the gap would
	// put it at 4 s.
	video := toneVideo(t, 6, "between(t,1,2)+between(t,4,5)", "asetpts='if(gte(T,3),PTS+1/TB,PTS)'")
	asr := &burstASR{}
	segments, err := sourcetimeline.New(asr, t.TempDir()).Transcribe(context.Background(), video, sourcetimeline.Window{})
	if err != nil {
		t.Fatal(err)
	}
	assertStarts(t, segments, 1, 5)
}

func TestTrimWindowExtractsOnlyTheWindowAndClips(t *testing.T) {
	requireFFmpeg(t)
	// Bursts at 6 s (inside 5-8) and 8.5 s (past the trim end).
	video := toneVideo(t, 10, "between(t,6,6.5)+between(t,8.5,9)", "")
	asr := &burstASR{}
	segments, err := sourcetimeline.New(asr, t.TempDir()).Transcribe(context.Background(), video, sourcetimeline.Window{Start: 5, End: 8})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(asr.duration-3) > tolerance {
		t.Fatalf("extracted %.3fs of audio, want the 3s trim window", asr.duration)
	}
	assertStarts(t, segments, 6)
}

func TestTrimWindowWithAudioOffset(t *testing.T) {
	requireFFmpeg(t)
	// Audio starts 0.5 s late; tone at 2 s of audio = 2.5 s source time.
	video := offsetAudio(t, toneVideo(t, 5, "between(t,2,3)", ""), 0.5)
	asr := &burstASR{}
	segments, err := sourcetimeline.New(asr, t.TempDir()).Transcribe(context.Background(), video, sourcetimeline.Window{Start: 1, End: 4})
	if err != nil {
		t.Fatal(err)
	}
	assertStarts(t, segments, 2.5)
}

func TestScratchAudioIsRemoved(t *testing.T) {
	requireFFmpeg(t)
	video := toneVideo(t, 2, "gte(t,0.5)", "")
	dir := t.TempDir()
	for _, asr := range []*burstASR{{}, {err: errors.New("worker down")}} {
		_, _ = sourcetimeline.New(asr, dir).Transcribe(context.Background(), video, sourcetimeline.Window{})
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			t.Fatalf("scratch file %s left behind", e.Name())
		}
	}
}

func TestASRFailureIsReturned(t *testing.T) {
	requireFFmpeg(t)
	video := toneVideo(t, 2, "gte(t,0.5)", "")
	want := errors.New("worker down")
	_, err := sourcetimeline.New(&burstASR{err: want}, t.TempDir()).Transcribe(context.Background(), video, sourcetimeline.Window{})
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want wrapped %v", err, want)
	}
}

func TestMissingVideoFails(t *testing.T) {
	requireFFmpeg(t)
	_, err := sourcetimeline.New(&burstASR{}, t.TempDir()).Transcribe(context.Background(), filepath.Join(t.TempDir(), "nope.mp4"), sourcetimeline.Window{})
	if err == nil {
		t.Fatal("expected an extraction error")
	}
}
