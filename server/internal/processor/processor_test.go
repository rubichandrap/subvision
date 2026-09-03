package processor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/editspec"
	"github.com/rubichandrap/subvision/server/internal/transcriber"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"
)

type fakeStore struct {
	downloads   map[string]string
	downloadErr error
}

func (f *fakeStore) Download(ctx context.Context, key, destPath string) error {
	if f.downloadErr != nil {
		return f.downloadErr
	}
	if f.downloads == nil {
		f.downloads = map[string]string{}
	}
	f.downloads[key] = destPath
	return nil
}

type fakePublisher struct {
	jobs []vfxjob.Job
	err  error
}

func (f *fakePublisher) Publish(job vfxjob.Job) error {
	if f.err != nil {
		return f.err
	}
	f.jobs = append(f.jobs, job)
	return nil
}

func newTestProcessor(pub *fakePublisher, store ObjectStore, transcribe TranscribeFunc) *Processor {
	proc := New(Options{
		Publisher:        pub,
		Store:            store,
		Transcribe:       transcribe,
		TmpDir:           "tmp",
		WhisperModelPath: "model.bin",
	})
	proc.convert = func(inputPath, outputPath string, window [2]float64) error {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(outputPath, []byte("pcm"), 0o644)
	}
	return proc
}

func TestProcessUploadedFileTranscribesTrimWindowOnly(t *testing.T) {
	var gotWindow [2]float64
	pub := &fakePublisher{}
	store := &fakeStore{}
	proc := newTestProcessor(pub, store, func(modelPath, audioPath string) ([]transcriber.Segment, error) {
		// The fake transcription returns times local to the trim window,
		// exactly what whisper produces for a sliced wav.
		return []transcriber.Segment{
			{Start: 0.2, End: 1.4, Text: "hello"},
			{Start: 1.6, End: 2.9, Text: "there"},
		}, nil
	})
	proc.convert = func(inputPath, outputPath string, window [2]float64) error {
		gotWindow = window
		return os.WriteFile(outputPath, []byte("pcm"), 0o644)
	}

	spec, err := editspec.Parse(`{"trim":{"start":30,"end":41},"frame":{"preset":"9:16","ratio":0.5625,"zoom":1,"panX":0,"panY":0},"animation":"fade"}`)
	if err != nil {
		t.Fatalf("parse fixture edit spec: %v", err)
	}

	if err := proc.ProcessUploadedFile("u1", "uploads/u1", spec); err != nil {
		t.Fatalf("ProcessUploadedFile: %v", err)
	}

	if gotWindow[0] != 30 || gotWindow[1] != 41 {
		t.Fatalf("convert window = %v, want [30 41] (transcribe must stay inside the trim)", gotWindow)
	}
	job := pub.jobs[0]
	// Segments shift back to absolute source time for the vfx contract.
	if job.Segments[0].Start != 30.2 || job.Segments[0].End != 31.4 || job.Segments[1].Start != 31.6 || job.Segments[1].End != 32.9 {
		t.Errorf("segments must shift back by trim.start 30, got %+v", job.Segments)
	}
}

func TestProcessUploadedFilePublishesVfxJob(t *testing.T) {
	segments := []transcriber.Segment{
		{Start: 0, End: 1.5, Text: "hello"},
		{Start: 1.5, End: 3, Text: "world"},
	}
	pub := &fakePublisher{}
	store := &fakeStore{}
	proc := newTestProcessor(pub, store, func(modelPath, audioPath string) ([]transcriber.Segment, error) {
		return segments, nil
	})

	spec, err := editspec.Parse(`{"trim":{"start":2,"end":9},"frame":{"preset":"9:16","ratio":0.5625,"zoom":1,"panX":0,"panY":0},"animation":"karaoke","style":{"fontFamily":"Inter","fontSizeScale":1,"color":"#FFFFFF","outlineWidth":0,"outlineColor":"#000000","bottomMargin":0.12,"background":"none","backgroundOpacity":0.5,"uppercase":false,"highlightColor":"#FACC15"}}`)
	if err != nil {
		t.Fatalf("parse fixture edit spec: %v", err)
	}

	if err := proc.ProcessUploadedFile("u1", "uploads/u1", spec); err != nil {
		t.Fatalf("ProcessUploadedFile: %v", err)
	}

	if len(pub.jobs) != 1 {
		t.Fatalf("expected exactly one published vfx job, got %d", len(pub.jobs))
	}
	job := pub.jobs[0]
	if job.UploadID != "u1" {
		t.Errorf("UploadID = %q, want %q", job.UploadID, "u1")
	}
	if job.ObjectKey != "uploads/u1" {
		t.Errorf("ObjectKey = %q, want %q", job.ObjectKey, "uploads/u1")
	}
	if len(job.Segments) != 2 || job.Segments[0].Text != "hello" || job.Segments[1].Text != "world" {
		t.Errorf("segments did not reach the vfx job unchanged: %+v", job.Segments)
	}
	if job.EditSpec != spec {
		t.Errorf("the Edit Spec did not reach the vfx job: %+v", job.EditSpec)
	}
	if dest, ok := store.downloads["uploads/u1"]; !ok {
		t.Errorf("video was not downloaded from object key uploads/u1 (downloads: %v)", store.downloads)
	} else if filepath.Dir(dest) != filepath.Join("tmp", "videos") {
		t.Errorf("video downloaded to %q, want it under the videos temp dir", dest)
	}
}

func TestProcessUploadedFilePublishErrorSurfaces(t *testing.T) {
	pub := &fakePublisher{err: errors.New("broker down")}
	proc := newTestProcessor(pub, &fakeStore{}, func(string, string) ([]transcriber.Segment, error) {
		return nil, nil
	})

	err := proc.ProcessUploadedFile("u1", "uploads/u1", nil)
	if err == nil || !strings.Contains(err.Error(), "broker down") {
		t.Fatalf("expected the publish error to surface, got %v", err)
	}
}

func TestProcessUploadedFileRejectsUnexpectedObjectKey(t *testing.T) {
	pub := &fakePublisher{}
	proc := newTestProcessor(pub, &fakeStore{}, func(string, string) ([]transcriber.Segment, error) {
		return nil, nil
	})

	err := proc.ProcessUploadedFile("u1", "no-prefix/u1", nil)
	if err == nil {
		t.Fatal("expected an error for an object key outside the uploads prefix")
	}
	if len(pub.jobs) != 0 {
		t.Errorf("no vfx job should be published for an unexpected key, got %d", len(pub.jobs))
	}
}
