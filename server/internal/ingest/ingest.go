package ingest

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/editspec"
	"github.com/rubichandrap/subvision/server/internal/transcript"
)

// UploadJob carries the upload identifier, object key in object storage,
// and raw Edit Spec metadata from tus upload metadata.
type UploadJob struct {
	UploadID    string
	ObjectKey   string
	RawEditSpec string
}

// VideoDownloader downloads a video from object storage to a local path.
type VideoDownloader interface {
	Download(ctx context.Context, key, destPath string) error
}


// AudioExtractor extracts 16kHz mono WAV audio from a video file within an optional trim window.
type AudioExtractor interface {
	ExtractAudio(ctx context.Context, inputPath, outputPath string, window [2]float64) error
}


// AudioTranscriber converts a WAV audio file into Transcription Segments.
type AudioTranscriber interface {
	Transcribe(audioPath string) ([]transcript.Segment, error)
}


// ProcessLifecycle coordinates stage transitions and atomic commit for an upload process.
type ProcessLifecycle interface {
	StartTranscription(uploadID string) (bool, error)
	CommitIngestion(ctx context.Context, id string, segments []transcript.Segment, spec *editspec.Spec) error
	MarkFailed(uploadID, reason string) (bool, error)
}

// FFmpegAudioExtractor extracts audio using the ffmpeg CLI.
type FFmpegAudioExtractor struct{}

func NewFFmpegAudioExtractor() *FFmpegAudioExtractor {
	return &FFmpegAudioExtractor{}
}

func (e *FFmpegAudioExtractor) ExtractAudio(ctx context.Context, inputPath, outputPath string, window [2]float64) error {
	args := []string{"-hide_banner"}
	if window[0] > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", window[0]))
	}
	args = append(args, "-i", inputPath, "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1")
	if window[1] > 0 {
		args = append(args, "-to", fmt.Sprintf("%.3f", window[1]))
	}
	args = append(args, outputPath)
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("[ffmpeg] Running conversion command: %v", cmd.Args)
	return cmd.Run()
}

// Config configures the Ingestion pipeline with adapters and scratch directory.
type Config struct {
	VideoDownloader  VideoDownloader
	AudioExtractor   AudioExtractor
	AudioTranscriber AudioTranscriber
	ProcessLifecycle ProcessLifecycle
	TmpDir           string
}

// Pipeline owns the end-to-end ingestion workflow: validation, download,
// extraction, speech transcription, coordinate shifting, atomic commit,
// scratch file cleanup, and autonomous failure guards.
type Pipeline struct {
	downloader  VideoDownloader
	extractor   AudioExtractor
	transcriber AudioTranscriber
	lifecycle   ProcessLifecycle
	videoTmpDir string
	audioTmpDir string
}

// New creates an Ingestion pipeline with the supplied configuration.
func New(cfg Config) *Pipeline {
	tmpDir := cfg.TmpDir
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	videoDir := filepath.Join(tmpDir, "videos")
	audioDir := filepath.Join(tmpDir, "audios")
	_ = os.MkdirAll(videoDir, 0755)
	_ = os.MkdirAll(audioDir, 0755)

	return &Pipeline{
		downloader:  cfg.VideoDownloader,
		extractor:   cfg.AudioExtractor,
		transcriber: cfg.AudioTranscriber,
		lifecycle:   cfg.ProcessLifecycle,
		videoTmpDir: videoDir,
		audioTmpDir: audioDir,
	}
}

// ProcessUpload executes the ingestion pipeline for an upload job. Any error
// encountered automatically transitions the Process to failed before returning.
func (p *Pipeline) ProcessUpload(ctx context.Context, job UploadJob) (err error) {
	// Autonomous failure guard: any failure records the cause on the Process.
	defer func() {
		if err != nil && p.lifecycle != nil && job.UploadID != "" {
			if recorded, markErr := p.lifecycle.MarkFailed(job.UploadID, err.Error()); markErr != nil {
				log.Printf("[Ingest] Failed to mark process %s as failed: %v", job.UploadID, markErr)
			} else if !recorded {
				log.Printf("[Ingest] Process %s unknown or terminal, failure reason not recorded: %v", job.UploadID, err)
			}
		}
	}()

	if job.ObjectKey == "" {
		return errors.New("upload job carried no object key")
	}
	if !strings.HasPrefix(job.ObjectKey, config.ObjectPrefix) {
		return fmt.Errorf("unexpected object key %q: must start with %q", job.ObjectKey, config.ObjectPrefix)
	}

	spec, err := editspec.Parse(job.RawEditSpec)
	if err != nil {
		return fmt.Errorf("invalid edit spec: %w", err)
	}

	if p.lifecycle != nil {
		if recorded, err := p.lifecycle.StartTranscription(job.UploadID); err != nil {
			log.Printf("[Ingest] %v", err)
		} else if !recorded {
			log.Printf("[Ingest] lifecycle: job %s unknown or terminal, not marking transcribing", job.UploadID)
		}
	}

	id := strings.TrimPrefix(job.ObjectKey, config.ObjectPrefix)
	videoPath := filepath.Join(p.videoTmpDir, id)
	audioPath := filepath.Join(p.audioTmpDir, fmt.Sprintf("%s.wav", id))

	// Deferred scratch file cleanup ensures no temporary files leak on success or failure.
	defer func() {
		if videoPath != "" {
			_ = os.Remove(videoPath)
		}
		if audioPath != "" {
			_ = os.Remove(audioPath)
		}
	}()


	if p.downloader != nil {
		if err := p.downloader.Download(ctx, job.ObjectKey, videoPath); err != nil {
			return fmt.Errorf("failed to download video from object storage: %w", err)
		}
	}

	var window [2]float64
	if spec != nil {
		window = [2]float64{spec.Trim.Start, spec.Trim.End}
	}

	if p.extractor != nil {
		if err := p.extractor.ExtractAudio(ctx, videoPath, audioPath, window); err != nil {
			return fmt.Errorf("failed to extract audio: %w", err)
		}
	}

	var segments []transcript.Segment
	if p.transcriber != nil {
		var err error
		segments, err = p.transcriber.Transcribe(audioPath)
		if err != nil {
			return fmt.Errorf("failed to transcribe audio: %w", err)
		}
	}

	if window[0] > 0 {
		segments = transcript.Shift(segments, window[0])
	}

	if p.lifecycle != nil {
		if err := p.lifecycle.CommitIngestion(ctx, job.UploadID, segments, spec); err != nil {
			return fmt.Errorf("failed to commit ingestion: %w", err)
		}
	}

	return nil
}
