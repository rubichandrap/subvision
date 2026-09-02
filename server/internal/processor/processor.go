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
	"github.com/rubichandrap/subvision/server/internal/transcriber"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"
)

// VfxJobPublisher publishes VFX Jobs; implemented by the rabbitmq publisher
// and faked in tests.
type VfxJobPublisher interface {
	Publish(job vfxjob.Job) error
}

type ObjectStore interface {
	Upload(ctx context.Context, key, filePath string) error
	Download(ctx context.Context, key, destPath string) error
}

// TranscribeFunc converts an audio file into Transcription Segments; the
// whisper-backed implementation is wired in main.
type TranscribeFunc func(modelPath, audioPath string) ([]transcriber.Segment, error)

// ConvertFunc extracts a wav from a video file; the ffmpeg-backed
// implementation is wired in New.
type ConvertFunc func(inputPath, outputPath string) error

type Processor struct {
	publisher        VfxJobPublisher
	store            ObjectStore
	transcribe       TranscribeFunc
	convert          ConvertFunc
	videoTmpDir      string
	audioTmpDir      string
	whisperModelPath string
}

func New(publisher VfxJobPublisher, store ObjectStore, transcribe TranscribeFunc, tmpDir, whisperModelPath string) *Processor {
	return &Processor{
		publisher:        publisher,
		store:            store,
		transcribe:       transcribe,
		convert:          convertToWav,
		videoTmpDir:      filepath.Join(tmpDir, "videos"),
		audioTmpDir:      filepath.Join(tmpDir, "audios"),
		whisperModelPath: whisperModelPath,
	}
}

// ProcessUploadedFile runs an upload through the pipeline: download the video,
// extract and transcribe its audio, then publish a VFX Job carrying the
// Transcription Segments.
func (p *Processor) ProcessUploadedFile(uploadID, objectKey string) error {
	ctx := context.Background()
	log.Printf("[Processor] Start processing upload %s (object %s)", uploadID, objectKey)

	if !strings.HasPrefix(objectKey, config.ObjectPrefix) {
		return fmt.Errorf("unexpected object key %q: must start with %q", objectKey, config.ObjectPrefix)
	}
	id := strings.TrimPrefix(objectKey, config.ObjectPrefix)

	videoPath := filepath.Join(p.videoTmpDir, id)
	if err := p.store.Download(ctx, objectKey, videoPath); err != nil {
		return fmt.Errorf("failed to download video from object storage: %w", err)
	}
	log.Printf("[Processor] Downloaded video to %s", videoPath)

	audioPath := filepath.Join(p.audioTmpDir, fmt.Sprintf("%s.wav", id))
	if err := p.convert(videoPath, audioPath); err != nil {
		return fmt.Errorf("failed to convert to wav: %w", err)
	}
	log.Printf("[Processor] Converted to WAV: %s", audioPath)

	segments, err := p.transcribe(p.whisperModelPath, audioPath)
	if err != nil {
		return fmt.Errorf("failed to transcribe audio: %w", err)
	}
	log.Printf("[Processor] Transcribed %d segments", len(segments))

	job := vfxjob.Job{
		UploadID:  uploadID,
		ObjectKey: objectKey,
		Segments:  segments,
	}
	if err := p.publisher.Publish(job); err != nil {
		return fmt.Errorf("failed to publish vfx job for upload %s: %w", uploadID, err)
	}
	log.Printf("[Processor] Published vfx job for upload %s", uploadID)

	return nil
}

func convertToWav(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("[ffmpeg] Running conversion command: %v", cmd.Args)
	return cmd.Run()
}
