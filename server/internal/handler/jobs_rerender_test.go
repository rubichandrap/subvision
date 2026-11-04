package handler

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rubichandrap/subvision/server/internal/db"
	"github.com/rubichandrap/subvision/server/internal/job"
	"github.com/rubichandrap/subvision/server/internal/transcript"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"
)

// fakeRerenderPublisher is a no-op VFX publisher that always succeeds,
// satisfying job.VfxPublisher.
type fakeRerenderPublisher struct{}

func (f *fakeRerenderPublisher) Publish(_ vfxjob.Job) error { return nil }

// newRerenderRouter builds a router backed by a real in-memory job.Store for
// smoke testing the rerender and segment-save routes.
func newRerenderRouter(t *testing.T) (*gin.Engine, *job.Store) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	pub := &fakeRerenderPublisher{}
	outputs := &fakeOutputs{body: "video bytes"}
	store, err := job.NewStore(database, pub, &fakeCleaner{})
	if err != nil {
		t.Fatalf("create job store: %v", err)
	}
	router := gin.New()
	RegisterJobs(router, store, outputs)
	return router, store
}

// TestRerenderRouteIsRegistered verifies the POST /jobs/:id/rerender route
// exists and returns 404 (not 405 Method Not Allowed) for unknown jobs,
// confirming the route is wired.
func TestRerenderRouteIsRegistered(t *testing.T) {
	router, _ := newRerenderRouter(t)
	rec := doPost(t, router, "/jobs/missing/rerender")
	if rec.Code != http.StatusNotFound {
		t.Errorf("POST /jobs/missing/rerender = %d, want 404 (route must be registered)", rec.Code)
	}
}

// TestSaveSegmentsRouteIsRegistered verifies the PUT /jobs/:id/segments route
// exists and returns 400 for a bad body (not 405), confirming the route is wired.
func TestSaveSegmentsRouteIsRegistered(t *testing.T) {
	router, _ := newRerenderRouter(t)
	rec := doPut(t, router, "/jobs/missing/segments", `{}`)
	// {} has no segments field → 400 bad request before the store is hit
	if rec.Code != http.StatusBadRequest {
		t.Errorf("PUT /jobs/missing/segments = %d, want 400 (route must be registered)", rec.Code)
	}
}

// TestRerenderReturns409ForNonDoneJob verifies the handler maps
// ErrStageConflict → 409 Conflict.
func TestRerenderReturns409ForNonDoneJob(t *testing.T) {
	router, store := newRerenderRouter(t)
	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	rec := doPost(t, router, "/jobs/u1/rerender")
	if rec.Code != http.StatusConflict {
		t.Errorf("POST /jobs/u1/rerender (uploaded stage) = %d, want 409", rec.Code)
	}
}

// TestSaveSegmentsReturns409ForUneditableStage verifies the handler maps
// ErrStageConflict → 409 when segments are saved on an in-flight (non-editable) job.
func TestSaveSegmentsReturns409ForUneditableStage(t *testing.T) {
	router, store := newRerenderRouter(t)
	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	// uploaded stage is not editable
	const body = `{"segments":[{"start":1.0,"end":2.0,"text":"x","words":[]}]}`
	rec := doPut(t, router, "/jobs/u1/segments", body)
	if rec.Code != http.StatusConflict {
		t.Errorf("PUT /jobs/u1/segments (uploaded stage) = %d, want 409", rec.Code)
	}
}

// TestSaveSegmentsNullClearsTranscript verifies that a JSON null segments
// value is treated as an empty transcript rather than crashing the client
// on a null payload.
func TestSaveSegmentsNullClearsTranscript(t *testing.T) {
	router, store := newRerenderRouter(t)
	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.CommitIngestion(context.Background(), "u1", nil, nil); err != nil {
		t.Fatalf("commit ingestion: %v", err)
	}
	rec := doPut(t, router, "/jobs/u1/segments", `{"segments":null}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /jobs/u1/segments (null) = %d: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"segments":[]`) {
		t.Errorf("response = %s, want an empty segments array", rec.Body)
	}
}

// TestSaveSegmentsReturns400ForTextWordCountChange verifies user-correctable
// sync failures are returned as a 400 with the reason, not a bare 500.
func TestSaveSegmentsReturns400ForTextWordCountChange(t *testing.T) {
	router, store := newRerenderRouter(t)
	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	original := []transcript.Segment{{
		Start: 1,
		End:   2,
		Text:  "hello world",
		Words: []transcript.Word{
			{Text: "hello", Start: 1, End: 1.4},
			{Text: "world", Start: 1.5, End: 2},
		},
	}}
	if err := store.CommitIngestion(context.Background(), "u1", original, nil); err != nil {
		t.Fatalf("commit ingestion: %v", err)
	}
	const body = `{"segments":[{"start":1,"end":2,"text":"hello brave new world","words":[]}],"sourceIndices":[0]}`
	rec := doPut(t, router, "/jobs/u1/segments", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT /jobs/u1/segments (word-count change) = %d: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "want 2 timed words") {
		t.Errorf("response = %s, want the text/word-count reason", rec.Body)
	}
}

// TestSaveSegmentsReturns400ForBadTiming verifies the handler maps
// transcript.ValidationError → 400 Bad Request.
func TestSaveSegmentsReturns400ForBadTiming(t *testing.T) {
	router, store := newRerenderRouter(t)
	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.CommitIngestion(context.Background(), "u1", nil, nil); err != nil {
		t.Fatalf("commit ingestion: %v", err)
	}
	// negative start → ValidationError
	const body = `{"segments":[{"start":-1,"end":2,"text":"x","words":[]}]}`
	rec := doPut(t, router, "/jobs/u1/segments", body)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("PUT /jobs/u1/segments (bad timing) = %d, want 400", rec.Code)
	}
}
