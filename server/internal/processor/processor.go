package processor

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/editspec"
	"github.com/rubichandrap/subvision/server/internal/job"
	"github.com/rubichandrap/subvision/server/internal/transcriber"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"
)

// VfxJobPublisher publishes VFX Jobs; implemented by the rabbitmq publisher
// and faked in tests.
type VfxJobPublisher interface {
	Publish(job vfxjob.Job) error
}

type ObjectStore interface {
	Download(ctx context.Context, key, destPath string) error
}

// TranscribeFunc converts an audio file into Transcription Segments; the
// whisper-backed implementation is wired in main.
type TranscribeFunc func(modelPath, audioPath string) ([]transcriber.Segment, error)

// ConvertFunc extracts a wav from a video file; the ffmpeg-backed
// implementation is wired in New.
type ConvertFunc func(inputPath, outputPath string, window [2]float64) error

type Options struct {
	Publisher        VfxJobPublisher
	Store            ObjectStore
	Transcribe       TranscribeFunc
	TmpDir           string
	WhisperModelPath string
	Lifecycle        job.Tracker // optional
}

type Processor struct {
	publisher        VfxJobPublisher
	store            ObjectStore
	transcribe       TranscribeFunc
	convert          ConvertFunc
	lifecycle        job.Tracker
	videoTmpDir      string
	audioTmpDir      string
	whisperModelPath string
}

func New(opts Options) *Processor {
	return &Processor{
		publisher:        opts.Publisher,
		store:            opts.Store,
		transcribe:       opts.Transcribe,
		convert:          convertToWav,
		lifecycle:        opts.Lifecycle,
		videoTmpDir:      filepath.Join(opts.TmpDir, "videos"),
		audioTmpDir:      filepath.Join(opts.TmpDir, "audios"),
		whisperModelPath: opts.WhisperModelPath,
	}
}

// ProcessUploadedFile runs an upload through the pipeline: download the video,
// extract and transcribe its audio, then publish a VFX Job carrying the
// Transcription Segments and the upload's Edit Spec (nil when the upload had
// none).
func (p *Processor) ProcessUploadedFile(uploadID, objectKey string, spec *editspec.Spec) error {
	ctx := context.Background()
	log.Printf("[Processor] Start processing upload %s (object %s)", uploadID, objectKey)
	if p.lifecycle != nil {
		recorded, err := p.lifecycle.MarkTranscribing(uploadID)
		if err != nil {
			log.Printf("[Processor] %v", err)
		} else if !recorded {
			log.Printf("[Processor] lifecycle: job %s unknown or terminal, not marking %s", uploadID, "transcribing")
		}
	}

	if !strings.HasPrefix(objectKey, config.ObjectPrefix) {
		return fmt.Errorf("unexpected object key %q: must start with %q", objectKey, config.ObjectPrefix)
	}
	id := strings.TrimPrefix(objectKey, config.ObjectPrefix)

	videoPath := filepath.Join(p.videoTmpDir, id)
	if err := p.store.Download(ctx, objectKey, videoPath); err != nil {
		return fmt.Errorf("failed to download video from object storage: %w", err)
	}
	log.Printf("[Processor] Downloaded video to %s", videoPath)

	// Whisper only ever hears the trim window: captioning an 11-second Edit
	// must not transcribe a 56-minute source. The window is [start, end] in
	// source seconds, [0, 0] meaning the whole video. Segment times come
	// back local to the window and are shifted back to absolute source time
	// before publishing — word timings shift with their segment — so the
	// vfx contract (absolute in) stays untouched.
	var window [2]float64
	if spec != nil {
		window = [2]float64{spec.Trim.Start, spec.Trim.End}
	}

	audioPath := filepath.Join(p.audioTmpDir, fmt.Sprintf("%s.wav", id))
	if err := p.convert(videoPath, audioPath, window); err != nil {
		return fmt.Errorf("failed to convert to wav: %w", err)
	}
	log.Printf("[Processor] Converted to WAV: %s (window %.3f-%.3f)", audioPath, window[0], window[1])

	segments, err := p.transcribe(p.whisperModelPath, audioPath)
	if err != nil {
		return fmt.Errorf("failed to transcribe audio: %w", err)
	}
	log.Printf("[Processor] Transcribed %d segments", len(segments))

	if window[0] > 0 {
		for i := range segments {
			segments[i].Start += window[0]
			segments[i].End += window[0]
			for j := range segments[i].Words {
				segments[i].Words[j].Start += window[0]
				segments[i].Words[j].End += window[0]
			}
		}
	}

	job := vfxjob.Job{
		UploadID:  uploadID,
		ObjectKey: objectKey,
		Segments:  segments,
		EditSpec:  spec,
	}
	if err := p.publisher.Publish(job); err != nil {
		return fmt.Errorf("failed to publish vfx job for upload %s: %w", uploadID, err)
	}
	log.Printf("[Processor] Published vfx job for upload %s", uploadID)
	if p.lifecycle != nil {
		recorded, err := p.lifecycle.MarkRendering(uploadID)
		if err != nil {
			log.Printf("[Processor] %v", err)
		} else if !recorded {
			log.Printf("[Processor] lifecycle: job %s unknown or terminal, not marking %s", uploadID, "rendering")
		}
	}

	return nil
}

func convertToWav(inputPath, outputPath string, window [2]float64) error {
	args := []string{"-hide_banner"}
	// Input seeking before -i is a fast stream-level seek; the trailing -to
	// bounds the read end. -to is absolute source time, so it composes with
	// -ss correctly for end != 0.
	if window[0] > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", window[0]))
	}
	args = append(args, "-i", inputPath, "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1")
	if window[1] > 0 {
		args = append(args, "-to", fmt.Sprintf("%.3f", window[1]))
	}
	args = append(args, outputPath)
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("[ffmpeg] Running conversion command: %v", cmd.Args)
	return cmd.Run()
}
