package job

import (
	"context"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/db"
	"github.com/rubichandrap/subvision/server/internal/editspec"
)

func TestReopenMovesDoneToRendering(t *testing.T) {
	handle, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { handle.Close() })
	store, err := NewStore(handle, nil, nil)
	if err != nil {
		t.Fatalf("create job store: %v", err)
	}

	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if recorded, err := store.MarkDone("u1", "outputs/u1"); err != nil || !recorded {
		t.Fatalf("mark done: recorded=%v err=%v", recorded, err)
	}

	reopened, err := store.Reopen("u1")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if !reopened {
		t.Fatal("reopen of a done job must take effect")
	}
	got, err := store.Get("u1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Stage != StageRendering {
		t.Errorf("stage = %q, want %q", got.Stage, StageRendering)
	}
}

func TestReopenRefusesNonDone(t *testing.T) {
	handle, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { handle.Close() })
	store, err := NewStore(handle, nil, nil)
	if err != nil {
		t.Fatalf("create job store: %v", err)
	}

	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if recorded, err := store.MarkTranscribing("u1"); err != nil || !recorded {
		t.Fatalf("mark transcribing: recorded=%v err=%v", recorded, err)
	}
	if reopened, err := store.Reopen("u1"); err != nil || reopened {
		t.Errorf("reopen of an in-flight job must not take effect: reopened=%v err=%v", reopened, err)
	}

	if err := store.Create("u2", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if recorded, err := store.MarkFailed("u2", "boom"); err != nil || !recorded {
		t.Fatalf("mark failed: recorded=%v err=%v", recorded, err)
	}
	if reopened, err := store.Reopen("u2"); err != nil || reopened {
		t.Errorf("reopen of a failed job must not take effect: reopened=%v err=%v", reopened, err)
	}

	if reopened, err := store.Reopen("missing"); err != nil || reopened {
		t.Errorf("reopen of an unknown id must not take effect: reopened=%v err=%v", reopened, err)
	}
}

func TestEditSpecRoundTrip(t *testing.T) {
	handle, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { handle.Close() })
	store, err := NewStore(handle, &fakePublisher{}, nil)
	if err != nil {
		t.Fatalf("create job store: %v", err)
	}

	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}

	const raw = `{"trim":{"start":2,"end":9},"frame":{"preset":"9:16","ratio":0.5625,"zoom":1,"panX":0,"panY":0},"animation":"karaoke"}`
	spec, err := editspec.Parse(raw)
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	if err := store.CommitIngestion(context.Background(), "u1", nil, spec); err != nil {
		t.Fatalf("commit ingestion: %v", err)
	}
	got, err := store.EditSpec("u1")
	if err != nil {
		t.Fatalf("read edit spec: %v", err)
	}
	parsedGot, err := editspec.Parse(got)
	if err != nil {
		t.Fatalf("parse got spec: %v", err)
	}
	if parsedGot.Animation != spec.Animation || parsedGot.Trim != spec.Trim {
		t.Errorf("edit spec = %+v, want %+v", parsedGot, spec)
	}
}

func TestEditSpecEmptyWhenNoneStored(t *testing.T) {
	handle, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { handle.Close() })
	store, err := NewStore(handle, nil, nil)
	if err != nil {
		t.Fatalf("create job store: %v", err)
	}

	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := store.EditSpec("u1")
	if err != nil {
		t.Fatalf("read edit spec: %v", err)
	}
	if got != "" {
		t.Errorf("edit spec = %q, want empty", got)
	}
}
