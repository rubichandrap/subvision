package job

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/editspec"
	"github.com/rubichandrap/subvision/server/internal/transcript"
)

func TestCommitIngestionSuccess(t *testing.T) {
	pub := &fakePublisher{}
	cleaner := &fakeCleaner{}
	store := newTestStoreWithPorts(t, pub, cleaner)
	id := "u-commit-success"
	createJobWithStage(t, store, id, StageTranscribing)

	segments := []transcript.Segment{
		{
			Start: 1.0,
			End:   2.5,
			Text:  "hello world",
			Words: []transcript.Word{
				{Text: "hello", Start: 1.1, End: 1.7},
				{Text: "world", Start: 1.8, End: 2.4},
			},
		},
	}
	spec := &editspec.Spec{
		Trim: editspec.Trim{Start: 1.0, End: 5.0},
		Frame: editspec.Frame{
			Preset: "9:16",
			Ratio:  0.5625,
			Zoom:   1.0,
		},
		Animation: "karaoke",
		Style: &editspec.Style{
			FontFamily:        "Montserrat",
			FontSizeScale:     1.0,
			Color:             "#FFFFFF",
			OutlineWidth:      2.0,
			OutlineColor:      "#000000",
			BottomMargin:      0.1,
			Background:        "none",
			BackgroundOpacity: 0.0,
			Uppercase:         false,
			HighlightColor:    "#FFFF00",
		},
	}
	ctx := context.Background()
	err := store.CommitIngestion(ctx, id, segments, spec)
	if err != nil {
		t.Fatalf("commit ingestion failed: %v", err)
	}

	// Verify process transitioned to rendering
	proc, err := store.Get(id)
	if err != nil {
		t.Fatalf("get process: %v", err)
	}
	if proc.Stage != StageRendering {
		t.Errorf("process stage = %q, want %q", proc.Stage, StageRendering)
	}

	// Verify segments were persisted
	storedSegmentsRaw, err := store.Segments(id)
	if err != nil {
		t.Fatalf("read segments: %v", err)
	}
	var storedSegments []transcript.Segment
	if err := json.Unmarshal([]byte(storedSegmentsRaw), &storedSegments); err != nil {
		t.Fatalf("unmarshal stored segments: %v", err)
	}
	if len(storedSegments) != 1 || storedSegments[0].Text != "hello world" {
		t.Errorf("unexpected stored segments: %+v", storedSegments)
	}

	// Verify edit spec was persisted
	storedSpecRaw, err := store.EditSpec(id)
	if err != nil {
		t.Fatalf("read edit spec: %v", err)
	}
	storedSpec, err := editspec.Parse(storedSpecRaw)
	if err != nil {
		t.Fatalf("parse stored edit spec: %v", err)
	}
	if storedSpec == nil || storedSpec.Animation != "karaoke" {
		t.Errorf("unexpected stored edit spec: %+v", storedSpec)
	}

	// Verify VFX job was published
	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published VFX job, got %d", len(pub.published))
	}
	vfx := pub.published[0]
	if vfx.UploadID != id {
		t.Errorf("published uploadId = %q, want %q", vfx.UploadID, id)
	}
	if vfx.ObjectKey != "uploads/"+id {
		t.Errorf("published objectKey = %q, want %q", vfx.ObjectKey, "uploads/"+id)
	}
	if len(vfx.Segments) != 1 || vfx.Segments[0].Text != "hello world" {
		t.Errorf("unexpected published segments: %+v", vfx.Segments)
	}
	if vfx.EditSpec == nil || vfx.EditSpec.Animation != "karaoke" {
		t.Errorf("unexpected published edit spec: %+v", vfx.EditSpec)
	}
}

func TestCommitIngestionClearsEditSpecWhenNil(t *testing.T) {
	pub := &fakePublisher{}
	cleaner := &fakeCleaner{}
	store := newTestStoreWithPorts(t, pub, cleaner)
	id := "u-commit-clears-spec"
	createJobWithStage(t, store, id, StageTranscribing)

	// Pre-populate an edit spec
	specJSON := `{"animation":"karaoke"}`
	if err := store.SaveEditSpec(id, specJSON); err != nil {
		t.Fatalf("save initial edit spec: %v", err)
	}

	segments := []transcript.Segment{
		{Start: 0, End: 1, Text: "sample"},
	}

	ctx := context.Background()
	// Commit with nil spec
	err := store.CommitIngestion(ctx, id, segments, nil)
	if err != nil {
		t.Fatalf("commit ingestion: %v", err)
	}

	// Verify edit spec is cleared in store
	storedSpecRaw, err := store.EditSpec(id)
	if err != nil {
		t.Fatalf("read edit spec: %v", err)
	}
	if storedSpecRaw != "" {
		t.Errorf("expected empty stored edit spec, got %q", storedSpecRaw)
	}

	// Verify published job has nil EditSpec
	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published VFX job, got %d", len(pub.published))
	}
	if pub.published[0].EditSpec != nil {
		t.Errorf("expected nil published EditSpec, got %+v", pub.published[0].EditSpec)
	}
}

func TestCommitIngestionRejectsTerminalJobs(t *testing.T) {
	terminalStages := []Stage{StageDone, StageFailed}
	for _, stage := range terminalStages {
		t.Run(string(stage), func(t *testing.T) {
			pub := &fakePublisher{}
			cleaner := &fakeCleaner{}
			store := newTestStoreWithPorts(t, pub, cleaner)
			id := "u-terminal-" + string(stage)
			createJobWithStage(t, store, id, stage)

			segments := []transcript.Segment{
				{Start: 1.0, End: 2.0, Text: "should not commit"},
			}
			err := store.CommitIngestion(context.Background(), id, segments, nil)
			if err == nil {
				t.Fatalf("expected error committing to terminal stage %s, got nil", stage)
			}
			if !errors.Is(err, ErrStageConflict) {
				t.Fatalf("expected ErrStageConflict, got %v", err)
			}
			var conflictErr *StageConflictError
			if !errors.As(err, &conflictErr) {
				t.Fatalf("expected *StageConflictError, got %T", err)
			}
			if conflictErr.Stage != stage {
				t.Errorf("conflict error stage = %q, want %q", conflictErr.Stage, stage)
			}

			// Verify state was not corrupted
			p, err := store.Get(id)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if p.Stage != stage {
				t.Errorf("process stage changed to %q, want %q", p.Stage, stage)
			}

			// Verify no segments were persisted
			segs, err := store.Segments(id)
			if err != nil {
				t.Fatalf("read segments: %v", err)
			}
			if segs != "" {
				t.Errorf("expected empty segments, got %q", segs)
			}

			// Verify publisher was not invoked
			if len(pub.published) != 0 {
				t.Errorf("expected 0 published jobs, got %d", len(pub.published))
			}
		})
	}
}

func TestCommitIngestionRejectsUnknownJob(t *testing.T) {
	pub := &fakePublisher{}
	cleaner := &fakeCleaner{}
	store := newTestStoreWithPorts(t, pub, cleaner)

	segments := []transcript.Segment{
		{Start: 1.0, End: 2.0, Text: "unknown"},
	}
	err := store.CommitIngestion(context.Background(), "non-existent-id", segments, nil)
	if err == nil {
		t.Fatal("expected error for unknown job, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if len(pub.published) != 0 {
		t.Errorf("expected 0 published jobs, got %d", len(pub.published))
	}
}

func TestCommitIngestionRollsBackToFailedOnPublishError(t *testing.T) {
	publishErr := errors.New("rabbitmq down")
	pub := &fakePublisher{err: publishErr}
	cleaner := &fakeCleaner{}
	store := newTestStoreWithPorts(t, pub, cleaner)
	id := "u-commit-publish-fail"
	createJobWithStage(t, store, id, StageTranscribing)

	segments := []transcript.Segment{
		{Start: 0.5, End: 1.5, Text: "test segments"},
	}

	err := store.CommitIngestion(context.Background(), id, segments, nil)
	if err == nil {
		t.Fatal("expected error on failed publish, got nil")
	}
	if !strings.Contains(err.Error(), "rabbitmq down") {
		t.Errorf("expected error mentioning rabbitmq down, got %v", err)
	}

	// Verify process transitioned to failed
	p, err := store.Get(id)
	if err != nil {
		t.Fatalf("get process: %v", err)
	}
	if p.Stage != StageFailed {
		t.Errorf("process stage = %q, want %q", p.Stage, StageFailed)
	}
	if !strings.Contains(p.Reason, "rabbitmq down") {
		t.Errorf("process reason = %q, want it to contain rabbitmq down", p.Reason)
	}

	// Verify segments were still saved so transcript is not lost
	stored, err := store.Segments(id)
	if err != nil {
		t.Fatalf("read segments: %v", err)
	}
	if stored == "" {
		t.Error("expected segments to remain saved after publish failure")
	}
}

func TestCommitIngestionPublishesObjectKeyWithCompositeTusdId(t *testing.T) {
	pub := &fakePublisher{}
	cleaner := &fakeCleaner{}
	store := newTestStoreWithPorts(t, pub, cleaner)
	compositeID := "c031d87a4149fa8617ba8d8fecff003e+4_u8Gf0Vxyz"
	expectedObjectID := "c031d87a4149fa8617ba8d8fecff003e"
	createJobWithStage(t, store, compositeID, StageTranscribing)

	segments := []transcript.Segment{
		{Start: 1.0, End: 2.0, Text: "composite test"},
	}

	err := store.CommitIngestion(context.Background(), compositeID, segments, nil)
	if err != nil {
		t.Fatalf("commit ingestion: %v", err)
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

func TestCommitIngestionNilPublisherDoesNotStrandJob(t *testing.T) {
	store := newTestStoreWithPorts(t, nil, &fakeCleaner{})
	id := "u-commit-nil-pub"
	createJobWithStage(t, store, id, StageTranscribing)

	segments := []transcript.Segment{
		{Start: 1.0, End: 2.0, Text: "nil pub test"},
	}

	err := store.CommitIngestion(context.Background(), id, segments, nil)
	if err == nil {
		t.Fatal("expected error with nil publisher, got nil")
	}

	// Job must not have moved out of transcribing
	p, getErr := store.Get(id)
	if getErr != nil {
		t.Fatalf("get: %v", getErr)
	}
	if p.Stage != StageTranscribing {
		t.Errorf("stage = %q after nil-publisher commit, want %q (must not strand in rendering)", p.Stage, StageTranscribing)
	}
}
