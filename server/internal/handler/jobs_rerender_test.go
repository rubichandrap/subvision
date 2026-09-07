package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/rubichandrap/subvision/server/internal/db"
	"github.com/rubichandrap/subvision/server/internal/job"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"
)

type fakeRerenderPublisher struct {
	jobs []vfxjob.Job
	err  error
}

func (f *fakeRerenderPublisher) Publish(job vfxjob.Job) error {
	if f.err != nil {
		return f.err
	}
	f.jobs = append(f.jobs, job)
	return nil
}

var errBrokerDown = errors.New("broker down")

func newRerenderRouter(t *testing.T) (*gin.Engine, *job.Store, *fakeRerenderPublisher) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	pub := &fakeRerenderPublisher{}
	outputs := &fakeOutputs{body: "video bytes"}
	cleaner := &fakeCleaner{}
	store, err := job.NewStore(database, pub, cleaner)
	if err != nil {
		t.Fatalf("create job store: %v", err)
	}
	router := gin.New()
	RegisterJobs(router, store, store, store, pub, outputs, cleaner)
	return router, store, pub
}

func doPut(t *testing.T, router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func doPost(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(nil))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func createDoneJob(t *testing.T, store *job.Store, id string) {
	t.Helper()
	if err := store.Create(id, "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	const stored = `[{"start":1.5,"end":2.5,"text":"hello","words":[{"text":"hello","start":1.6,"end":2.4}]}]`
	if err := store.SaveOriginalSegments(id, stored); err != nil {
		t.Fatalf("save segments: %v", err)
	}
	if recorded, err := store.MarkRendering(id); err != nil || !recorded {
		t.Fatalf("mark rendering: recorded=%v err=%v", recorded, err)
	}
	if recorded, err := store.MarkDone(id, "outputs/"+id); err != nil || !recorded {
		t.Fatalf("mark done: recorded=%v err=%v", recorded, err)
	}
}

func TestSaveSegmentsPersistsEdits(t *testing.T) {
	router, store, _ := newRerenderRouter(t)
	createDoneJob(t, store, "u1")

	const edited = `{"segments":[{"start":1.0,"end":2.0,"text":"fixed","words":[]}]}`
	rec := doPut(t, router, "/jobs/u1/segments", edited)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /jobs/u1/segments = %d: %s", rec.Code, rec.Body)
	}

	got, err := store.Segments("u1")
	if err != nil {
		t.Fatalf("read segments: %v", err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("stored segments not JSON: %v", err)
	}
	if len(decoded) != 1 || decoded[0]["text"] != "fixed" {
		t.Errorf("stored segments = %s, want the edited text", got)
	}
}

func TestSaveSegmentsRescalesWordOffsets(t *testing.T) {
	router, store, _ := newRerenderRouter(t)
	createDoneJob(t, store, "u1")

	// Original window 1.5-2.5 with word 1.6-2.4; edited window 1.0-3.0
	// must keep the word at its relative offset, scaled: 1.2-2.8.
	const edited = `{"segments":[{"start":1.0,"end":3.0,"text":"hello","words":[{"text":"hello","start":9.9,"end":9.9}]}]}`
	if rec := doPut(t, router, "/jobs/u1/segments", edited); rec.Code != http.StatusOK {
		t.Fatalf("PUT /jobs/u1/segments = %d: %s", rec.Code, rec.Body)
	}
	got, err := store.Segments("u1")
	if err != nil {
		t.Fatalf("read segments: %v", err)
	}
	var decoded []struct {
		Words []struct {
			Start float64 `json:"start"`
			End   float64 `json:"end"`
		} `json:"words"`
	}
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("stored segments not JSON: %v", err)
	}
	if len(decoded) != 1 || len(decoded[0].Words) != 1 {
		t.Fatalf("stored segments = %s, want one segment with one word", got)
	}
	w := decoded[0].Words[0]
	const wantStart, wantEnd = 1.2, 2.8
	if diff := w.Start - wantStart; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("word start = %v, want %v (relative offset scaled)", w.Start, wantStart)
	}
	if diff := w.End - wantEnd; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("word end = %v, want %v (relative offset scaled)", w.End, wantEnd)
	}
}

func TestSaveSegmentsRejectsBadTiming(t *testing.T) {
	router, store, _ := newRerenderRouter(t)
	createDoneJob(t, store, "u1")

	for name, body := range map[string]string{
		"negative start":   `{"segments":[{"start":-1,"end":2,"text":"x","words":[]}]}`,
		"end before start": `{"segments":[{"start":2,"end":1,"text":"x","words":[]}]}`,
		"unordered":        `{"segments":[{"start":0,"end":5,"text":"a","words":[]},{"start":1,"end":2,"text":"b","words":[]}]}`,
		"non-finite":       `{"segments":[{"start":"soon","end":2,"text":"x","words":[]}]}`,
		"malformed json":   `{"segments":[}`,
	} {
		rec := doPut(t, router, "/jobs/u1/segments", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: PUT = %d, want 400 (%s)", name, rec.Code, rec.Body)
		}
	}
}

func TestSaveSegmentsUnknownJobReturns404(t *testing.T) {
	router, _, _ := newRerenderRouter(t)

	rec := doPut(t, router, "/jobs/missing/segments", `{"segments":[]}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("PUT /jobs/missing/segments = %d, want 404", rec.Code)
	}
}

func TestRerenderPublishesJobWithEditedSegments(t *testing.T) {
	router, store, pub := newRerenderRouter(t)
	createDoneJob(t, store, "u1")

	const spec = `{"trim":{"start":2,"end":9},"frame":{"preset":"9:16","ratio":0.5625,"zoom":1,"panX":0,"panY":0},"animation":"karaoke"}`
	if err := store.SaveEditSpec("u1", spec); err != nil {
		t.Fatalf("save edit spec: %v", err)
	}
	const edited = `{"segments":[{"start":1.0,"end":2.0,"text":"fixed","words":[]}]}`
	if rec := doPut(t, router, "/jobs/u1/segments", edited); rec.Code != http.StatusOK {
		t.Fatalf("PUT segments = %d: %s", rec.Code, rec.Body)
	}

	rec := doPost(t, router, "/jobs/u1/rerender")
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /jobs/u1/rerender = %d: %s", rec.Code, rec.Body)
	}

	got, err := store.Get("u1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Stage != job.StageRendering {
		t.Errorf("stage = %q, want rendering", got.Stage)
	}

	if len(pub.jobs) != 1 {
		t.Fatalf("expected one published vfx job, got %d", len(pub.jobs))
	}
	rerender := pub.jobs[0]
	if rerender.UploadID != "u1" {
		t.Errorf("UploadID = %q, want u1", rerender.UploadID)
	}
	if rerender.ObjectKey != "uploads/u1" {
		t.Errorf("ObjectKey = %q, want uploads/u1", rerender.ObjectKey)
	}
	if len(rerender.Segments) != 1 || rerender.Segments[0].Text != "fixed" {
		t.Errorf("rerender must carry edited segments: %+v", rerender.Segments)
	}
	if rerender.EditSpec == nil || rerender.EditSpec.Animation != "karaoke" {
		t.Errorf("rerender must carry the original edit spec: %+v", rerender.EditSpec)
	}
}

func TestRerenderWithoutEditsRepublishesStoredSegments(t *testing.T) {
	router, store, pub := newRerenderRouter(t)
	createDoneJob(t, store, "u1")

	if rec := doPost(t, router, "/jobs/u1/rerender"); rec.Code != http.StatusOK {
		t.Fatalf("POST /jobs/u1/rerender = %d: %s", rec.Code, rec.Body)
	}
	if len(pub.jobs) != 1 {
		t.Fatalf("expected one published vfx job, got %d", len(pub.jobs))
	}
	rerender := pub.jobs[0]
	if len(rerender.Segments) != 1 || rerender.Segments[0].Text != "hello" {
		t.Errorf("unedited re-render must carry the stored segments: %+v", rerender.Segments)
	}
	if rerender.EditSpec != nil {
		t.Errorf("unedited re-render without a spec must carry nil spec: %+v", rerender.EditSpec)
	}
	if got, err := store.Get("u1"); err != nil || got.Stage != job.StageRendering {
		t.Errorf("stage must be rendering after re-render (got %+v, err %v)", got, err)
	}
}

func TestRerenderPublishFailureFailsJob(t *testing.T) {
	router, store, pub := newRerenderRouter(t)
	createDoneJob(t, store, "u1")
	pub.err = errBrokerDown

	if rec := doPost(t, router, "/jobs/u1/rerender"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("POST /jobs/u1/rerender = %d, want 500 (%s)", rec.Code, rec.Body)
	}
	got, err := store.Get("u1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Stage != job.StageFailed {
		t.Errorf("stage = %q, want failed (never stranded in rendering)", got.Stage)
	}
	if got.Reason == "" {
		t.Error("failed re-render must surface its reason")
	}
}

func TestRerenderRefusesNonDone(t *testing.T) {
	router, store, pub := newRerenderRouter(t)
	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}

	if rec := doPost(t, router, "/jobs/u1/rerender"); rec.Code != http.StatusConflict {
		t.Errorf("POST rerender of in-flight job = %d, want 409", rec.Code)
	}
	if rec := doPost(t, router, "/jobs/missing/rerender"); rec.Code != http.StatusNotFound {
		t.Errorf("POST rerender of unknown job = %d, want 404", rec.Code)
	}
	if len(pub.jobs) != 0 {
		t.Errorf("refused re-render must not publish, got %d jobs", len(pub.jobs))
	}
}
