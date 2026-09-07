package job

import (
	"testing"

	"github.com/rubichandrap/subvision/server/internal/db"
)

func newSegmentsTestStore(t *testing.T) *Store {
	t.Helper()
	handle, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { handle.Close() })
	store, err := NewStore(handle)
	if err != nil {
		t.Fatalf("create job store: %v", err)
	}
	return store
}

const segmentsFixture = `[{"start":30.2,"end":31.4,"text":"hello","words":[{"text":"hello","start":30.3,"end":31.3}]},{"start":31.6,"end":32.9,"text":"there","words":[]}]`

func TestSegmentsRoundTrip(t *testing.T) {
	store := newSegmentsTestStore(t)
	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := store.SaveSegments("u1", segmentsFixture); err != nil {
		t.Fatalf("save segments: %v", err)
	}

	got, err := store.Segments("u1")
	if err != nil {
		t.Fatalf("read segments: %v", err)
	}
	if got != segmentsFixture {
		t.Errorf("segments = %q, want %q", got, segmentsFixture)
	}

	// Segments outlive the queue message: still readable after done.
	if recorded, err := store.MarkDone("u1", "outputs/u1"); err != nil || !recorded {
		t.Fatalf("mark done: recorded=%v err=%v", recorded, err)
	}
	got, err = store.Segments("u1")
	if err != nil {
		t.Fatalf("read segments after done: %v", err)
	}
	if got != segmentsFixture {
		t.Errorf("segments after done = %q, want %q", got, segmentsFixture)
	}
}

func TestSegmentsEmptyWhenNoneStored(t *testing.T) {
	store := newSegmentsTestStore(t)
	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := store.Segments("u1")
	if err != nil {
		t.Fatalf("read segments: %v", err)
	}
	if got != "" {
		t.Errorf("segments = %q, want empty", got)
	}
}

func TestDeleteRemovesSegments(t *testing.T) {
	store := newSegmentsTestStore(t)
	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.SaveSegments("u1", segmentsFixture); err != nil {
		t.Fatalf("save segments: %v", err)
	}

	if deleted, err := store.Delete("u1"); err != nil || !deleted {
		t.Fatalf("delete: deleted=%v err=%v", deleted, err)
	}

	got, err := store.Segments("u1")
	if err != nil {
		t.Fatalf("read segments after delete: %v", err)
	}
	if got != "" {
		t.Errorf("segments after delete = %q, want empty", got)
	}
}
