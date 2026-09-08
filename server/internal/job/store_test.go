package job

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/db"
	"github.com/rubichandrap/subvision/server/internal/transcript"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"
)

type fakePublisher struct {
	published []vfxjob.Job
	err       error
}

func (f *fakePublisher) Publish(job vfxjob.Job) error {
	if f.err != nil {
		return f.err
	}
	f.published = append(f.published, job)
	return nil
}

type fakeCleaner struct {
	deleted []string
	err     error
}

func (f *fakeCleaner) Delete(ctx context.Context, prefix string) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, prefix)
	return nil
}

func newTestStoreWithPorts(t *testing.T, pub VfxPublisher, cleaner ObjectCleaner) *Store {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	store, err := NewStore(database, pub, cleaner)
	if err != nil {
		t.Fatalf("create job store: %v", err)
	}
	return store
}

func createJobWithStage(t *testing.T, store *Store, id string, stage Stage) {
	t.Helper()
	if err := store.Create(id, "video.mp4"); err != nil {
		t.Fatalf("create job: %v", err)
	}
	switch stage {
	case StageUploaded:
		// already uploaded
	case StageTranscribing:
		if _, err := store.MarkTranscribing(id); err != nil {
			t.Fatalf("mark transcribing: %v", err)
		}
	case StageRendering:
		if _, err := store.MarkRendering(id); err != nil {
			t.Fatalf("mark rendering: %v", err)
		}
	case StageDone:
		if _, err := store.MarkDone(id, "outputs/"+id); err != nil {
			t.Fatalf("mark done: %v", err)
		}
	case StageFailed:
		if _, err := store.MarkFailed(id, "pipeline error"); err != nil {
			t.Fatalf("mark failed: %v", err)
		}
	}
}

func TestSaveSegmentsRejectsUneditableStages(t *testing.T) {
	uneditableStages := []Stage{StageUploaded, StageTranscribing, StageFailed}
	for _, stage := range uneditableStages {
		t.Run(string(stage), func(t *testing.T) {
			store := newTestStoreWithPorts(t, &fakePublisher{}, &fakeCleaner{})
			id := "job-" + string(stage)
			createJobWithStage(t, store, id, stage)

			segments := []transcript.Segment{
				{Start: 1.0, End: 2.0, Text: "hello"},
			}
			_, err := store.SaveSegments(id, segments)
			if err == nil {
				t.Fatalf("expected error saving segments in stage %s, got nil", stage)
			}
			var conflictErr *StageConflictError
			if !errors.As(err, &conflictErr) {
				t.Fatalf("expected StageConflictError, got %T (%v)", err, err)
			}
			if conflictErr.Stage != stage {
				t.Errorf("conflict error stage = %q, want %q", conflictErr.Stage, stage)
			}
			if conflictErr.ID != id {
				t.Errorf("conflict error id = %q, want %q", conflictErr.ID, id)
			}
		})
	}
}

func TestSaveSegmentsAllowsRenderingAndDone(t *testing.T) {
	for _, stage := range []Stage{StageRendering, StageDone} {
		t.Run(string(stage), func(t *testing.T) {
			store := newTestStoreWithPorts(t, &fakePublisher{}, &fakeCleaner{})
			id := "job-" + string(stage)
			createJobWithStage(t, store, id, stage)

			segments := []transcript.Segment{
				{Start: 1.0, End: 2.5, Text: "valid segment"},
			}
			saved, err := store.SaveSegments(id, segments)
			if err != nil {
				t.Fatalf("unexpected error saving segments in stage %s: %v", stage, err)
			}
			if len(saved) != 1 || saved[0].Text != "valid segment" {
				t.Fatalf("unexpected saved segments: %+v", saved)
			}
		})
	}
}

func TestSaveSegmentsValidatesTimings(t *testing.T) {
	store := newTestStoreWithPorts(t, &fakePublisher{}, &fakeCleaner{})
	id := "job-invalid-timing"
	createJobWithStage(t, store, id, StageDone)

	// End before Start violates timing invariants
	segments := []transcript.Segment{
		{Start: 2.5, End: 1.0, Text: "backward time"},
	}
	_, err := store.SaveSegments(id, segments)
	if err == nil {
		t.Fatal("expected validation error for invalid timing, got nil")
	}
	var valErr *transcript.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *transcript.ValidationError, got %T (%v)", err, err)
	}
}

func TestSaveSegmentsRescalesWordsAgainstWhisperOriginals(t *testing.T) {
	store := newTestStoreWithPorts(t, &fakePublisher{}, &fakeCleaner{})
	id := "job-rescale"
	createJobWithStage(t, store, id, StageDone)

	// Store whisper original segment with timed words
	originalJSON := `[{"start":1.0,"end":2.0,"text":"hello world","words":[{"start":1.1,"end":1.5,"text":"hello"},{"start":1.6,"end":1.9,"text":"world"}]}]`
	if err := store.SaveOriginalSegments(id, originalJSON); err != nil {
		t.Fatalf("save original segments: %v", err)
	}

	// Edit segment bounds from [1.0, 2.0] to [2.0, 4.0] (stretched 2x, shifted +1.0)
	edited := []transcript.Segment{
		{Start: 2.0, End: 4.0, Text: "hello world"},
	}
	saved, err := store.SaveSegments(id, edited)
	if err != nil {
		t.Fatalf("save segments: %v", err)
	}
	if len(saved) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(saved))
	}
	if len(saved[0].Words) != 2 {
		t.Fatalf("expected 2 rescaled words, got %d", len(saved[0].Words))
	}

	// Original hello [1.1, 1.5] scaled into [2.0, 4.0] -> relative offset 0.1/1.0 = 10% -> 2.0 + 0.2 = 2.2
	w0 := saved[0].Words[0]
	if w0.Text != "hello" || w0.Start < 2.19 || w0.Start > 2.21 {
		t.Errorf("word 0 start = %f, want ~2.2", w0.Start)
	}
}

func TestRerenderVerifiesDoneStage(t *testing.T) {
	nonDoneStages := []Stage{StageUploaded, StageTranscribing, StageRendering, StageFailed}
	for _, stage := range nonDoneStages {
		t.Run(string(stage), func(t *testing.T) {
			store := newTestStoreWithPorts(t, &fakePublisher{}, &fakeCleaner{})
			id := "job-" + string(stage)
			createJobWithStage(t, store, id, stage)

			_, err := store.Rerender(id)
			if err == nil {
				t.Fatalf("expected error for re-rendering stage %s, got nil", stage)
			}
			var conflictErr *StageConflictError
			if !errors.As(err, &conflictErr) {
				t.Fatalf("expected StageConflictError, got %T (%v)", err, err)
			}
			if conflictErr.Stage != stage {
				t.Errorf("conflict error stage = %q, want %q", conflictErr.Stage, stage)
			}
		})
	}
}

func TestRerenderPublishesFreshVfxJobAndTransitionsToRendering(t *testing.T) {
	pub := &fakePublisher{}
	cleaner := &fakeCleaner{}
	store := newTestStoreWithPorts(t, pub, cleaner)
	id := "u-rerender"
	createJobWithStage(t, store, id, StageDone)

	// Save segments and edit spec
	segmentsJSON := `[{"start":1.0,"end":2.0,"text":"caption","words":[]}]`
	if err := store.SaveOriginalSegments(id, segmentsJSON); err != nil {
		t.Fatalf("save segments: %v", err)
	}
	specJSON := `{"trim":{"start":0,"end":10},"frame":{"preset":"9:16","ratio":0.5625,"zoom":1,"panX":0,"panY":0},"animation":"karaoke"}`
	if err := store.SaveEditSpec(id, specJSON); err != nil {
		t.Fatalf("save edit spec: %v", err)
	}

	proc, err := store.Rerender(id)
	if err != nil {
		t.Fatalf("rerender: %v", err)
	}
	if proc.Stage != StageRendering {
		t.Errorf("process stage = %q, want %q", proc.Stage, StageRendering)
	}

	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published VFX job, got %d", len(pub.published))
	}
	job := pub.published[0]
	if job.UploadID != id {
		t.Errorf("published uploadID = %q, want %q", job.UploadID, id)
	}
	if job.ObjectKey != "uploads/"+id {
		t.Errorf("published objectKey = %q, want %q", job.ObjectKey, "uploads/"+id)
	}
	if len(job.Segments) != 1 || job.Segments[0].Text != "caption" {
		t.Errorf("unexpected published segments: %+v", job.Segments)
	}
	if job.EditSpec == nil || job.EditSpec.Animation != "karaoke" {
		t.Errorf("unexpected published edit spec: %+v", job.EditSpec)
	}
}
func TestRerenderPublishesObjectKeyWithCompositeTusdId(t *testing.T) {
	pub := &fakePublisher{}
	cleaner := &fakeCleaner{}
	store := newTestStoreWithPorts(t, pub, cleaner)
	// tusd s3store generates IDs formatted as objectId+multipartId
	compositeID := "c031d87a4149fa8617ba8d8fecff003e+4_u8Gf0Vxyz"
	expectedObjectID := "c031d87a4149fa8617ba8d8fecff003e"
	createJobWithStage(t, store, compositeID, StageDone)

	segmentsJSON := `[{"start":1.0,"end":2.0,"text":"caption","words":[]}]`
	if err := store.SaveOriginalSegments(compositeID, segmentsJSON); err != nil {
		t.Fatalf("save segments: %v", err)
	}

	proc, err := store.Rerender(compositeID)
	if err != nil {
		t.Fatalf("rerender: %v", err)
	}
	if proc.Stage != StageRendering {
		t.Errorf("process stage = %q, want %q", proc.Stage, StageRendering)
	}

	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published VFX job, got %d", len(pub.published))
	}
	job := pub.published[0]
	if job.UploadID != compositeID {
		t.Errorf("published uploadID = %q, want %q", job.UploadID, compositeID)
	}
	wantObjectKey := "uploads/" + expectedObjectID
	if job.ObjectKey != wantObjectKey {
		t.Errorf("published objectKey = %q, want %q", job.ObjectKey, wantObjectKey)
	}
}


func TestRerenderRollsBackToFailedOnPublishError(t *testing.T) {
	publishErr := errors.New("rabbitmq down")
	pub := &fakePublisher{err: publishErr}
	cleaner := &fakeCleaner{}
	store := newTestStoreWithPorts(t, pub, cleaner)
	id := "u-rerender-fail"
	createJobWithStage(t, store, id, StageDone)

	specJSON := `{"trim":{"start":0,"end":10},"frame":{"preset":"9:16","ratio":0.5625,"zoom":1,"panX":0,"panY":0},"animation":"karaoke"}`
	if err := store.SaveEditSpec(id, specJSON); err != nil {
		t.Fatalf("save edit spec: %v", err)
	}

	_, err := store.Rerender(id)
	if err == nil {
		t.Fatal("expected error on failed publish, got nil")
	}
	if !strings.Contains(err.Error(), "rabbitmq down") {
		t.Errorf("expected error to mention rabbitmq down, got %v", err)
	}

	// Verify the process was rolled back to failed
	p, getErr := store.Get(id)
	if getErr != nil {
		t.Fatalf("get: %v", getErr)
	}
	if p.Stage != StageFailed {
		t.Errorf("process stage = %q, want %q", p.Stage, StageFailed)
	}
	if !strings.Contains(p.Reason, "rabbitmq down") {
		t.Errorf("process reason = %q, want it to contain rabbitmq down", p.Reason)
	}
}

func TestDeleteRemovesDatabaseRowsThenCleansObjectsBestEffort(t *testing.T) {
	cleaner := &fakeCleaner{}
	store := newTestStoreWithPorts(t, &fakePublisher{}, cleaner)
	id := "u-delete"
	createJobWithStage(t, store, id, StageDone)

	segmentsJSON := `[{"start":1.0,"end":2.0,"text":"caption","words":[]}]`
	if err := store.SaveOriginalSegments(id, segmentsJSON); err != nil {
		t.Fatalf("save segments: %v", err)
	}
	specJSON := `{"animation":"karaoke"}`
	if err := store.SaveEditSpec(id, specJSON); err != nil {
		t.Fatalf("save edit spec: %v", err)
	}

	deleted, err := store.Delete(id)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}

	// DB rows must be gone
	if _, err := store.Get(id); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for deleted job, got %v", err)
	}
	if segs, _ := store.Segments(id); segs != "" {
		t.Errorf("expected empty segments for deleted job, got %q", segs)
	}
	if spec, _ := store.EditSpec(id); spec != "" {
		t.Errorf("expected empty edit spec for deleted job, got %q", spec)
	}

	// Cleaner must have been invoked for upload and output prefixes
	wantCleaned := []string{"uploads/" + id, "outputs/" + id}
	if len(cleaner.deleted) != 2 {
		t.Fatalf("expected 2 deleted prefixes, got %d (%v)", len(cleaner.deleted), cleaner.deleted)
	}
	for i, want := range wantCleaned {
		if cleaner.deleted[i] != want {
			t.Errorf("cleaned[%d] = %q, want %q", i, cleaner.deleted[i], want)
		}
	}
}
func TestDeleteCleansObjectsWithCompositeTusdId(t *testing.T) {
	cleaner := &fakeCleaner{}
	store := newTestStoreWithPorts(t, &fakePublisher{}, cleaner)
	compositeID := "c031d87a4149fa8617ba8d8fecff003e+4_u8Gf0Vxyz"
	expectedObjectID := "c031d87a4149fa8617ba8d8fecff003e"
	createJobWithStage(t, store, compositeID, StageDone)

	deleted, err := store.Delete(compositeID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}

	wantCleaned := []string{"uploads/" + expectedObjectID, "outputs/" + expectedObjectID}
	if len(cleaner.deleted) != 2 {
		t.Fatalf("expected 2 deleted prefixes, got %d (%v)", len(cleaner.deleted), cleaner.deleted)
	}
	for i, want := range wantCleaned {
		if cleaner.deleted[i] != want {
			t.Errorf("cleaned[%d] = %q, want %q", i, cleaner.deleted[i], want)
		}
	}
}


func TestDeleteSucceedsEvenWhenObjectCleanerFails(t *testing.T) {
	cleaner := &fakeCleaner{err: errors.New("s3 connection timeout")}
	store := newTestStoreWithPorts(t, &fakePublisher{}, cleaner)
	id := "u-delete-s3-fail"
	createJobWithStage(t, store, id, StageDone)

	deleted, err := store.Delete(id)
	if err != nil {
		t.Fatalf("delete must not fail when object cleaner errors: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}

	// Row is still deleted despite cleaner failure
	if _, err := store.Get(id); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRerenderNilPublisherDoesNotStrandJob(t *testing.T) {
	// Constructing a store without a publisher is a configuration error, but
	// it must not leave the process stranded in rendering — the check must
	// fire before Reopen so the job stays done.
	store := newTestStoreWithPorts(t, nil, &fakeCleaner{})
	id := "u-nil-publisher"
	createJobWithStage(t, store, id, StageDone)

	_, err := store.Rerender(id)
	if err == nil {
		t.Fatal("expected error with nil publisher, got nil")
	}

	// Job must not have moved out of done
	p, getErr := store.Get(id)
	if getErr != nil {
		t.Fatalf("get: %v", getErr)
	}
	if p.Stage != StageDone {
		t.Errorf("stage = %q after nil-publisher rerender, want %q (must not strand in rendering)", p.Stage, StageDone)
	}
}
