package transcriber

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/go-audio/wav"
)

type Word struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
	Words []Word  `json:"words"`
}

// Word timestamp probability thresholds, defaulting to the upstream
// whisper.cpp defaults (thold_pt and thold_ptsum are both 0.01f — see the
// --word-thold CLI flag). Values move only on measured evidence (ADR-0006).
const (
	DefaultTokenThreshold    float32 = 0.01
	DefaultTokenSumThreshold float32 = 0.01
)

// Speech-gating parameters, measured on the acceptance fixture (ADR-0007):
// ffmpeg silencedetect at −30 dB with a 0.5 s minimum found the fixture's
// voice onset at 3.326 s against a measured ≈3.33 s — quieter floors treat
// the sample's room-tone ramp as speech. Windows split only at silences of
// at least DefaultWindowSplitSilence: whisper handles shorter pauses inside
// a window natively (its timestamp tokens advance the seek past them), while
// windows that end at every pause invite the decoder to run past the window
// into the zero-padded tail of its mel chunk — measured on the fixture, a
// 2 s window leaked 2 s past its end and duplicated the next window's
// words. The minimum speech window mirrors whisper.cpp's own VAD default
// (min_speech_duration_ms = 250): shorter blips are not worth a decode
// pass. Values move only on measured evidence.
const (
	DefaultSilenceNoise       = "-30dB"
	DefaultSilenceMinDuration = 0.5
	DefaultWindowSplitSilence = 2.0
	DefaultMinSpeechWindow    = 0.25
)

// thresholdSetter is the decoder-context surface applyWordThresholds needs.
type thresholdSetter interface {
	SetTokenThreshold(float32)
	SetTokenSumThreshold(float32)
}

// applyWordThresholds wires the word timestamp thresholds into the decoder
// context through the existing Go binding setters.
func applyWordThresholds(ctx thresholdSetter, token, tokenSum float32) {
	ctx.SetTokenThreshold(token)
	ctx.SetTokenSumThreshold(tokenSum)
}

// silenceInterval is one detected silence, [start, end] in seconds; an
// infinite end means the audio ends inside the silence.
type silenceInterval struct {
	start, end float64
}

// speechWindow is one detected speech region, [start, end] seconds on the
// original audio timeline.
type speechWindow struct {
	start, end float64
}

var (
	silenceStartRe = regexp.MustCompile(`silence_start:\s*([0-9.]+)`)
	silenceEndRe   = regexp.MustCompile(`silence_end:\s*([0-9.]+)`)
)

// parseSilencedetect extracts the silence intervals from ffmpeg silencedetect
// output: start/end pairs, with a dangling start meaning the audio ends
// inside that silence.
func parseSilencedetect(output string) []silenceInterval {
	var silences []silenceInterval
	var pending float64
	hasPending := false
	for _, line := range strings.Split(output, "\n") {
		if m := silenceStartRe.FindStringSubmatch(line); m != nil {
			// Malformed output (a second start while one is open): keep the
			// open interval rather than silently losing it.
			if hasPending {
				continue
			}
			value, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				continue
			}
			pending, hasPending = value, true
			continue
		}
		if m := silenceEndRe.FindStringSubmatch(line); m != nil && hasPending {
			value, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				continue
			}
			silences = append(silences, silenceInterval{pending, value})
			hasPending = false
		}
	}
	if hasPending {
		silences = append(silences, silenceInterval{pending, math.Inf(1)})
	}
	return silences
}

// speechWindows derives the speech regions to decode: the complement of the
// detected silences within [0, audioDur]. Only silences of at least
// DefaultWindowSplitSilence become window boundaries — shorter pauses stay
// inside a window, where whisper handles them natively. Remaining windows
// below the minimum speech window are stripped: a blip is not worth a decode
// pass (ADR-0007).
func speechWindows(audioDur float64, silences []silenceInterval) []speechWindow {
	clamped := make([]silenceInterval, 0, len(silences))
	for _, s := range silences {
		start, end := max(s.start, 0), min(s.end, audioDur)
		if end-start >= DefaultWindowSplitSilence {
			clamped = append(clamped, silenceInterval{start, end})
		}
	}
	sort.Slice(clamped, func(i, j int) bool { return clamped[i].start < clamped[j].start })

	merged := make([]silenceInterval, 0, len(clamped))
	for _, s := range clamped {
		if n := len(merged); n > 0 && s.start <= merged[n-1].end {
			merged[n-1].end = max(merged[n-1].end, s.end)
			continue
		}
		merged = append(merged, s)
	}

	var windows []speechWindow
	cursor := 0.0
	for _, s := range merged {
		if w := (speechWindow{cursor, s.start}); w.end-w.start >= DefaultMinSpeechWindow {
			windows = append(windows, w)
		}
		cursor = s.end
	}
	if w := (speechWindow{cursor, audioDur}); w.end-w.start >= DefaultMinSpeechWindow {
		windows = append(windows, w)
	}
	return windows
}

// clampSegmentsToWindow keeps a window's decode inside its own bounds:
// whisper can run past a window's end into the zero-padded tail of its mel
// chunk, and words reported there belong to later windows or to silence.
// Segments are rebuilt from their surviving words, mirroring SplitSegments'
// text rebuild (ADR-0007).
func clampSegmentsToWindow(segments []Segment, w speechWindow) []Segment {
	clamped := make([]Segment, 0, len(segments))
	for _, seg := range segments {
		words := make([]Word, 0, len(seg.Words))
		for _, word := range seg.Words {
			if word.End <= w.start || word.Start >= w.end {
				continue
			}
			word.Start = max(word.Start, w.start)
			word.End = min(word.End, w.end)
			words = append(words, word)
		}
		if len(words) == 0 {
			continue
		}
		clamped = append(clamped, segmentFromWords(words))
	}
	return clamped
}

// detectSilences runs silence detection over a WAV file and returns the
// silence intervals it found. A package var so tests can fake the ffmpeg
// subprocess and CI stays hermetic (ADR-0007).
var detectSilences = detectSilencesWithFFmpeg

func detectSilencesWithFFmpeg(wavPath string) ([]silenceInterval, error) {
	filter := fmt.Sprintf("silencedetect=noise=%s:d=%v", DefaultSilenceNoise, DefaultSilenceMinDuration)
	cmd := exec.Command("ffmpeg", "-nostats", "-hide_banner", "-i", wavPath, "-af", filter, "-f", "null", "-")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := stderr.String()
		if len(detail) > 512 {
			detail = detail[len(detail)-512:]
		}
		return nil, fmt.Errorf("silence detection failed: %w: %s", err, strings.TrimSpace(detail))
	}
	return parseSilencedetect(stderr.String()), nil
}

// noSpeechDetected reports whether gated transcription found no speech at
// all: gating on and zero speech windows — transcribed empty with a warning,
// correct output, not a failure (ADR-0007).
func noSpeechDetected(settings Settings, windows []speechWindow) bool {
	return settings.GatingEnabled() && len(windows) == 0
}

// transcribes the audio file at audioPath using the whisper model and
// options carried in settings: token timestamps with word thresholds, and —
// when speech gating is on — decoding only the speech windows ffmpeg
// silencedetect finds (ADR-0007)
func Transcribe(settings Settings, audioPath string) ([]Segment, error) {
	// The wav is loaded first: gating needs the audio duration to cut the
	// speech windows.
	data, err := loadWavToFloat32(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load wav: %w", err)
	}
	audioDur := float64(len(data)) / whisper.SampleRate

	windows := []speechWindow{{start: 0, end: audioDur}}
	if settings.GatingEnabled() {
		silences, err := detectSilences(audioPath)
		if err != nil {
			return nil, err
		}
		windows = speechWindows(audioDur, silences)
	}

	if noSpeechDetected(settings, windows) {
		log.Printf("[Transcriber] no speech detected in %s; transcribed empty", audioPath)
		return nil, nil
	}

	model, err := whisper.New(settings.ModelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load whisper model: %w", err)
	}
	defer model.Close()

	var segments []Segment
	for _, w := range windows {
		windowSegments, err := transcribeWindow(model, settings, data, w)
		if err != nil {
			return nil, err
		}
		segments = append(segments, windowSegments...)
	}

	// Whisper emits few long segments; the renderer needs short ones —
	// split on speech pauses before publishing (see ADR-0005).
	return SplitSegments(segments), nil
}

// transcribeWindow decodes one speech window on a fresh context: the stock
// binding never resets its segment cursor between Process calls, so every
// window gets its own (cheap) context against the loaded model (ADR-0007).
func transcribeWindow(model whisper.Model, settings Settings, data []float32, w speechWindow) ([]Segment, error) {
	ctx, err := model.NewContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create whisper context: %w", err)
	}

	// Word timings are transcribed, not derived: token timestamps make
	// whisper time every token, which the words below are built from.
	applyWordThresholds(ctx, DefaultTokenThreshold, DefaultTokenSumThreshold)
	ctx.SetTokenTimestamps(true)

	if settings.GatingEnabled() {
		// Offset and duration only bound the decode range — the audio buffer
		// is never cut, so reported timings stay on the original timeline.
		ctx.SetOffset(time.Duration(w.start * float64(time.Second)))
		ctx.SetDuration(time.Duration((w.end - w.start) * float64(time.Second)))
	}

	if err := ctx.Process(data, nil, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to process audio: %w", err)
	}

	var segments []Segment
	for n := 0; ; n++ {
		seg, err := ctx.NextSegment()
		if err != nil {
			break
		}
		words, err := wordsFromTokens(seg)
		if err != nil {
			return nil, fmt.Errorf("transcription segment %d: %w", n, err)
		}
		segments = append(segments, Segment{
			Start: seg.Start.Seconds(),
			End:   seg.End.Seconds(),
			Text:  seg.Text,
			Words: words,
		})
	}

	if settings.GatingEnabled() {
		segments = clampSegmentsToWindow(segments, w)
	}
	return segments, nil
}

// wordsFromTokens groups a segment's whisper tokens into Words carrying
// per-word timings. A token whose text begins with a space begins a new word
// (segment text is the plain concatenation of token texts); continuations and
// the punctuation that follows extend the current word. Token times are
// clamped into the segment's window.
//
// A segment that carries text but not a single non-zero token timestamp means
// token timestamps were never computed — that is an error, because word
// timings must come from whisper, they are never guessed.
func wordsFromTokens(seg whisper.Segment) ([]Word, error) {
	segStart := seg.Start.Seconds()
	segEnd := seg.End.Seconds()
	words := []Word{}
	seenText := false
	timestamped := false

	for _, token := range seg.Tokens {
		// "[_" is whisper.cpp's own special-token marker ([_BEG_], [_TT_2_], …).
		if strings.HasPrefix(token.Text, "[_") {
			continue
		}
		text := strings.TrimSpace(token.Text)
		if text == "" {
			continue
		}
		seenText = true
		if token.Start != 0 || token.End != 0 {
			timestamped = true
		}

		start := clampSeconds(token.Start.Seconds(), segStart, segEnd)
		end := clampSeconds(token.End.Seconds(), segStart, segEnd)
		if end < start {
			end = start
		}

		if strings.HasPrefix(token.Text, " ") || len(words) == 0 {
			words = append(words, Word{Text: text, Start: start, End: end})
		} else {
			last := &words[len(words)-1]
			last.Text += text
			last.End = end
		}
	}

	if seenText && !timestamped && seg.End > 0 {
		return nil, fmt.Errorf(
			"whisper returned no token timestamps (word timings unavailable); is token_timestamps enabled?")
	}
	return words, nil
}

func clampSeconds(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// loads a WAV file and returns its audio data as a slice of float32
func loadWavToFloat32(path string) ([]float32, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	dec := wav.NewDecoder(fh)
	if !dec.IsValidFile() {
		return nil, fmt.Errorf("invalid wav file")
	}

	buf, err := dec.FullPCMBuffer()
	if err != nil {
		return nil, err
	}

	if dec.SampleRate != whisper.SampleRate {
		return nil, fmt.Errorf("unsupported sample rate: %d", dec.SampleRate)
	}
	if dec.NumChans != 1 {
		return nil, fmt.Errorf("unsupported number of channels: %d", dec.NumChans)
	}

	return buf.AsFloat32Buffer().Data, nil
}
