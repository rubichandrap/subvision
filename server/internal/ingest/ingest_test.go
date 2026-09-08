package ingest_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/editspec"
	"github.com/rubichandrap/subvision/server/internal/ingest"
	"github.com/rubichandrap/subvision/server/internal/transcript"
)

type fakeDownloader struct {
	downloadedKey  string
	destPath       string
	err            error
	createFile     bool
}

func (f *fakeDownloader) Download(ctx context.Context, key, destPath string) error {
	f.downloadedKey = key
	f.destPath = destPath
	if f.createFile {
		_ = os.WriteFile(destPath, []byte("fake video data"), 0644)
	}
	return f.err
}

type fakeExtractor struct {
	inputPath  string
	outputPath string
	window     [2]float64
	err        error
	createFile bool
}

func (f *fakeExtractor) ExtractAudio(ctx context.Context, inputPath, outputPath string, window [2]float64) error {
	f.inputPath = inputPath
	f.outputPath = outputPath
	f.window = window
	if f.createFile {
		_ = os.WriteFile(outputPath, []byte("fake wav data"), 0644)
	}
	return f.err
}

type fakeTranscriber struct {
	audioPath string
	segments  []transcript.Segment
	err       error
}

func (f *fakeTranscriber) Transcribe(audioPath string) ([]transcript.Segment, error) {
	f.audioPath = audioPath
	return f.segments, f.err
}

type fakeLifecycle struct {
	startedTranscription string
	committedID          string
	committedSegments    []transcript.Segment
	committedSpec        *editspec.Spec
	failedID             string
	failedReason         string
	commitErr            error
}

func (f *fakeLifecycle) StartTranscription(uploadID string) (bool, error) {
	f.startedTranscription = uploadID
	return true, nil
}

func (f *fakeLifecycle) CommitIngestion(ctx context.Context, id string, segments []transcript.Segment, spec *editspec.Spec) error {
	f.committedID = id
	f.committedSegments = segments
	f.committedSpec = spec
	return f.commitErr
}

func (f *fakeLifecycle) MarkFailed(uploadID, reason string) (bool, error) {
	f.failedID = uploadID
	f.failedReason = reason
	return true, nil
}

func TestProcessUploadSuccessWithoutTrim(t *testing.T) {
	tmpDir := t.TempDir()
	downloader := &fakeDownloader{createFile: true}
	extractor := &fakeExtractor{createFile: true}
	trans := &fakeTranscriber{
		segments: []transcript.Segment{
			{Start: 0.5, End: 2.0, Text: "hello world"},
		},
	}
	lifecycle := &fakeLifecycle{}

	pipeline := ingest.New(ingest.Config{
		VideoDownloader:  downloader,
		AudioExtractor:   extractor,
		AudioTranscriber: trans,
		ProcessLifecycle: lifecycle,
		TmpDir:           tmpDir,
	})

	uploadJob := ingest.UploadJob{
		UploadID:    "upload-123",
		ObjectKey:   "uploads/upload-123",
		RawEditSpec: "",
	}

	ctx := context.Background()
	err := pipeline.ProcessUpload(ctx, uploadJob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lifecycle.startedTranscription != "upload-123" {
		t.Errorf("expected startedTranscription = upload-123, got %q", lifecycle.startedTranscription)
	}
	if downloader.downloadedKey != "uploads/upload-123" {
		t.Errorf("expected downloaded key = uploads/upload-123, got %q", downloader.downloadedKey)
	}
	if extractor.window != [2]float64{0, 0} {
		t.Errorf("expected zero window, got %v", extractor.window)
	}
	if lifecycle.committedID != "upload-123" {
		t.Errorf("expected committedID = upload-123, got %q", lifecycle.committedID)
	}
	if len(lifecycle.committedSegments) != 1 || lifecycle.committedSegments[0].Start != 0.5 {
		t.Errorf("unexpected committed segments: %+v", lifecycle.committedSegments)
	}
	if lifecycle.committedSpec != nil {
		t.Errorf("expected nil committedSpec, got %+v", lifecycle.committedSpec)
	}

	// Verify scratch files were deleted
	if downloader.destPath != "" {
		if _, err := os.Stat(downloader.destPath); !os.IsNotExist(err) {
			t.Errorf("expected video file to be cleaned up at %s", downloader.destPath)
		}
	}
	if extractor.outputPath != "" {
		if _, err := os.Stat(extractor.outputPath); !os.IsNotExist(err) {
			t.Errorf("expected audio file to be cleaned up at %s", extractor.outputPath)
		}
	}
}

func TestProcessUploadSuccessWithTrimWindowShiftsSegments(t *testing.T) {
	tmpDir := t.TempDir()
	downloader := &fakeDownloader{createFile: true}
	extractor := &fakeExtractor{createFile: true}
	trans := &fakeTranscriber{
		segments: []transcript.Segment{
			{
				Start: 0.2,
				End:   1.4,
				Text:  "hello",
				Words: []transcript.Word{
					{Text: "hello", Start: 0.3, End: 1.3},
				},
			},
			{Start: 1.6, End: 2.9, Text: "there"},
		},
	}
	lifecycle := &fakeLifecycle{}

	pipeline := ingest.New(ingest.Config{
		VideoDownloader:  downloader,
		AudioExtractor:   extractor,
		AudioTranscriber: trans,
		ProcessLifecycle: lifecycle,
		TmpDir:           tmpDir,
	})

	rawSpec := `{"trim":{"start":30.0,"end":45.0},"frame":{"preset":"9:16","ratio":0.5625,"zoom":1,"panX":0,"panY":0},"animation":"karaoke"}`
	uploadJob := ingest.UploadJob{
		UploadID:    "upload-trim",
		ObjectKey:   "uploads/upload-trim",
		RawEditSpec: rawSpec,
	}

	ctx := context.Background()
	err := pipeline.ProcessUpload(ctx, uploadJob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Extractor must receive trim window
	expectedWindow := [2]float64{30.0, 45.0}
	if extractor.window != expectedWindow {
		t.Errorf("extractor window = %v, want %v", extractor.window, expectedWindow)
	}

	// Segments must be shifted by 30.0
	if len(lifecycle.committedSegments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(lifecycle.committedSegments))
	}
	s0 := lifecycle.committedSegments[0]
	if s0.Start != 30.2 || s0.End != 31.4 {
		t.Errorf("segment 0 timing = %f-%f, want 30.2-31.4", s0.Start, s0.End)
	}
	if len(s0.Words) != 1 || s0.Words[0].Start != 30.3 || s0.Words[0].End != 31.3 {
		t.Errorf("segment 0 word timing = %+v, want 30.3-31.3", s0.Words)
	}
	s1 := lifecycle.committedSegments[1]
	if s1.Start != 31.6 || s1.End != 32.9 {
		t.Errorf("segment 1 timing = %f-%f, want 31.6-32.9", s1.Start, s1.End)
	}

	// Spec must be committed
	if lifecycle.committedSpec == nil || lifecycle.committedSpec.Trim.Start != 30.0 {
		t.Errorf("unexpected committed spec: %+v", lifecycle.committedSpec)
	}
}

func TestProcessUploadAutonomousFailureGuard(t *testing.T) {
	testCases := []struct {
		name          string
		job           ingest.UploadJob
		downloader    *fakeDownloader
		extractor     *fakeExtractor
		transcriber   *fakeTranscriber
		lifecycle     *fakeLifecycle
		wantErrSubstr string
	}{
		{
			name: "missing object key",
			job: ingest.UploadJob{
				UploadID:    "u-err-1",
				ObjectKey:   "",
				RawEditSpec: "",
			},
			wantErrSubstr: "no object key",
		},
		{
			name: "invalid object key prefix",
			job: ingest.UploadJob{
				UploadID:    "u-err-2",
				ObjectKey:   "invalid/prefix/u-err-2",
				RawEditSpec: "",
			},
			wantErrSubstr: "must start with",
		},
		{
			name: "invalid edit spec JSON",
			job: ingest.UploadJob{
				UploadID:    "u-err-3",
				ObjectKey:   "uploads/u-err-3",
				RawEditSpec: "invalid-json",
			},
			wantErrSubstr: "edit spec",
		},
		{
			name: "downloader error",
			job: ingest.UploadJob{
				UploadID:    "u-err-4",
				ObjectKey:   "uploads/u-err-4",
				RawEditSpec: "",
			},
			downloader:    &fakeDownloader{err: errors.New("s3 connection failed")},
			wantErrSubstr: "s3 connection failed",
		},
		{
			name: "audio extractor error",
			job: ingest.UploadJob{
				UploadID:    "u-err-5",
				ObjectKey:   "uploads/u-err-5",
				RawEditSpec: "",
			},
			downloader:    &fakeDownloader{createFile: true},
			extractor:     &fakeExtractor{err: errors.New("ffmpeg exited 1")},
			wantErrSubstr: "ffmpeg exited 1",
		},
		{
			name: "transcription error",
			job: ingest.UploadJob{
				UploadID:    "u-err-6",
				ObjectKey:   "uploads/u-err-6",
				RawEditSpec: "",
			},
			downloader:    &fakeDownloader{createFile: true},
			extractor:     &fakeExtractor{createFile: true},
			transcriber:   &fakeTranscriber{err: errors.New("whisper cgo panic")},
			wantErrSubstr: "whisper cgo panic",
		},
		{
			name: "commit ingestion error",
			job: ingest.UploadJob{
				UploadID:    "u-err-7",
				ObjectKey:   "uploads/u-err-7",
				RawEditSpec: "",
			},
			downloader:    &fakeDownloader{createFile: true},
			extractor:     &fakeExtractor{createFile: true},
			transcriber:   &fakeTranscriber{segments: []transcript.Segment{{Start: 0, End: 1}}},
			lifecycle:     &fakeLifecycle{commitErr: errors.New("sqlite disk full")},
			wantErrSubstr: "sqlite disk full",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			dl := tc.downloader
			if dl == nil {
				dl = &fakeDownloader{createFile: true}
			}
			ext := tc.extractor
			if ext == nil {
				ext = &fakeExtractor{createFile: true}
			}
			tr := tc.transcriber
			if tr == nil {
				tr = &fakeTranscriber{segments: []transcript.Segment{{Start: 0, End: 1}}}
			}
			lc := tc.lifecycle
			if lc == nil {
				lc = &fakeLifecycle{}
			}

			p := ingest.New(ingest.Config{
				VideoDownloader:  dl,
				AudioExtractor:   ext,
				AudioTranscriber: tr,
				ProcessLifecycle: lc,
				TmpDir:           tmpDir,
			})

			err := p.ProcessUpload(context.Background(), tc.job)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErrSubstr)
			}
			if !strings.Contains(err.Error(), tc.wantErrSubstr) {
				t.Errorf("error = %q, want substr %q", err.Error(), tc.wantErrSubstr)
			}

			// Autonomous failure guard check: MarkFailed must have been called
			if lc.failedID != tc.job.UploadID {
				t.Errorf("lifecycle failedID = %q, want %q", lc.failedID, tc.job.UploadID)
			}
			if !strings.Contains(lc.failedReason, tc.wantErrSubstr) {
				t.Errorf("lifecycle failedReason = %q, want substr %q", lc.failedReason, tc.wantErrSubstr)
			}
		})
	}
}

func TestProcessUploadScratchFilesCleanedUpOnFailure(t *testing.T) {
	tmpDir := t.TempDir()
	dl := &fakeDownloader{createFile: true}
	ext := &fakeExtractor{createFile: true}
	tr := &fakeTranscriber{err: errors.New("transcribe failed")}
	lc := &fakeLifecycle{}

	p := ingest.New(ingest.Config{
		VideoDownloader:  dl,
		AudioExtractor:   ext,
		AudioTranscriber: tr,
		ProcessLifecycle: lc,
		TmpDir:           tmpDir,
	})

	uploadJob := ingest.UploadJob{
		UploadID:    "u-scratch-fail",
		ObjectKey:   "uploads/u-scratch-fail",
		RawEditSpec: "",
	}

	err := p.ProcessUpload(context.Background(), uploadJob)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify both scratch files were removed despite failure
	if dl.destPath == "" || ext.outputPath == "" {
		t.Fatal("destPath or outputPath was not recorded")
	}
	if _, err := os.Stat(dl.destPath); !os.IsNotExist(err) {
		t.Errorf("scratch video file still exists at %s", dl.destPath)
	}
	if _, err := os.Stat(ext.outputPath); !os.IsNotExist(err) {
		t.Errorf("scratch audio file still exists at %s", ext.outputPath)
	}
}
