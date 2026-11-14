package ingest

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/editspec"
	"github.com/rubichandrap/subvision/server/internal/sourcetimeline"
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

// SourceTranscriber turns a video's trim window into Transcription Segments
// on the Source Timeline, clipped to the window (sourcetimeline.Timeline).
type SourceTranscriber interface {
	Transcribe(ctx context.Context, videoPath string, window sourcetimeline.Window) ([]transcript.Segment, error)
}

// ProcessLifecycle coordinates stage transitions and atomic commit for an upload process.
type ProcessLifecycle interface {
	StartTranscription(uploadID string) (bool, error)
	CommitIngestion(ctx context.Context, id string, segments []transcript.Segment, spec *editspec.Spec) error
	MarkFailed(uploadID, reason string) (bool, error)
}

// Config configures the Ingestion pipeline with adapters and scratch directory.
type Config struct {
	VideoDownloader   VideoDownloader
	SourceTranscriber SourceTranscriber
	ProcessLifecycle  ProcessLifecycle
	TmpDir            string
}

// Pipeline owns the end-to-end ingestion workflow: validation, download,
// Source Timeline transcription, cue reflow, atomic commit,
// scratch file cleanup, and autonomous failure guards.
type Pipeline struct {
	downloader  VideoDownloader
	transcriber SourceTranscriber
	lifecycle   ProcessLifecycle
	videoTmpDir string
}

// New creates an Ingestion pipeline with the supplied configuration.
func New(cfg Config) *Pipeline {
	tmpDir := cfg.TmpDir
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	videoDir := filepath.Join(tmpDir, "videos")
	_ = os.MkdirAll(videoDir, 0755)

	return &Pipeline{
		downloader:  cfg.VideoDownloader,
		transcriber: cfg.SourceTranscriber,
		lifecycle:   cfg.ProcessLifecycle,
		videoTmpDir: videoDir,
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

	// Deferred scratch file cleanup ensures no temporary files leak on success or failure.
	defer func() { _ = os.Remove(videoPath) }()

	if p.downloader != nil {
		if err := p.downloader.Download(ctx, job.ObjectKey, videoPath); err != nil {
			return fmt.Errorf("failed to download video from object storage: %w", err)
		}
	}

	var window sourcetimeline.Window
	if spec != nil {
		window = sourcetimeline.Window{Start: spec.Trim.Start, End: spec.Trim.End}
	}

	// Segments come back on the Source Timeline, already clipped to the trim
	// window; ingestion never handles audio files or time offsets itself.
	var segments []transcript.Segment
	if p.transcriber != nil {
		var err error
		segments, err = p.transcriber.Transcribe(ctx, videoPath, window)
		if err != nil {
			return fmt.Errorf("failed to transcribe audio: %w", err)
		}
	}

	// Subtitle reflow is server-side: cue readability is not the ASR worker's
	// job. The existing Edit Spec already carries the user's `wordsPerPage`.
	segments = transcript.Reflow(segments, transcript.DefaultReflowOptions(wordsPerPage(spec)))

	// Cue-sized segments must stay monotonic for the renderer and the
	// editor; a misbehaving worker must fail loudly here, not persist.
	if err := transcript.ValidateTiming(segments); err != nil {
		return fmt.Errorf("invalid transcript after reflow: %w", err)
	}

	if p.lifecycle != nil {
		if err := p.lifecycle.CommitIngestion(ctx, job.UploadID, segments, spec); err != nil {
			return fmt.Errorf("failed to commit ingestion: %w", err)
		}
	}

	return nil
}

func wordsPerPage(spec *editspec.Spec) int {
	if spec == nil || spec.Captions == nil || spec.Captions.WordsPerPage <= 0 {
		return 4
	}
	return spec.Captions.WordsPerPage
}
