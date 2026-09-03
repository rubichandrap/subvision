package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rubichandrap/subvision/server/internal/db"
	"github.com/rubichandrap/subvision/server/internal/job"
)

type fakeOutputs struct {
	body    string
	openErr error
	opened  []string
}

func (f *fakeOutputs) Open(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	if f.openErr != nil {
		return nil, 0, f.openErr
	}
	f.opened = append(f.opened, key)
	return io.NopCloser(strings.NewReader(f.body)), int64(len(f.body)), nil
}

type fakeCleaner struct {
	deleted []string
}

func (f *fakeCleaner) Delete(ctx context.Context, prefix string) error {
	f.deleted = append(f.deleted, prefix)
	return nil
}

func newJobsRouter(t *testing.T) (*gin.Engine, *job.Store, *fakeOutputs, *fakeCleaner) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	store, err := job.NewStore(database)
	if err != nil {
		t.Fatalf("create job store: %v", err)
	}
	outputs := &fakeOutputs{body: "video bytes"}
	cleaner := &fakeCleaner{}
	router := gin.New()
	RegisterJobs(router, store, store, outputs, cleaner)
	return router, store, outputs, cleaner
}

func doGet(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func doDelete(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestLifecycleMovesThroughTheStatusAPI(t *testing.T) {
	router, store, _, _ := newJobsRouter(t)

	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}

	assertStage := func(want string) {
		t.Helper()
		rec := doGet(t, router, "/jobs/u1")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /jobs/u1 = %d: %s", rec.Code, rec.Body)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `"stage":"`+want+`"`) {
			t.Errorf("expected stage %q in %s", want, body)
		}
	}
	assertStage("uploaded")

	if recorded, err := store.MarkTranscribing("u1"); err != nil || !recorded {
		t.Fatalf("mark transcribing: recorded=%v err=%v", recorded, err)
	}
	assertStage("transcribing")

	if recorded, err := store.MarkRendering("u1"); err != nil || !recorded {
		t.Fatalf("mark rendering: recorded=%v err=%v", recorded, err)
	}
	assertStage("rendering")

	// not done yet: the response carries no download URL
	if rec := doGet(t, router, "/jobs/u1"); strings.Contains(rec.Body.String(), "downloadUrl") {
		t.Errorf("in-flight job must not offer a downloadUrl: %s", rec.Body)
	}

	if recorded, err := store.MarkDone("u1", "outputs/u1"); err != nil || !recorded {
		t.Fatalf("mark done: recorded=%v err=%v", recorded, err)
	}
	assertStage("done")

	rec := doGet(t, router, "/jobs/u1")
	if !strings.Contains(rec.Body.String(), `"downloadUrl":"/jobs/u1/download"`) {
		t.Errorf("done job must expose its download URL: %s", rec.Body)
	}

	download := doGet(t, router, "/jobs/u1/download")
	if download.Code != http.StatusOK {
		t.Fatalf("GET /jobs/u1/download = %d: %s", download.Code, download.Body)
	}
	if download.Body.String() != "video bytes" {
		t.Errorf("download body = %q, want the rendered output", download.Body.String())
	}
	if disposition := download.Header().Get("Content-Disposition"); !strings.Contains(disposition, "clip.mp4") {
		t.Errorf("Content-Disposition = %q, want the original filename", disposition)
	}
}

func TestFailedJobSurfacesItsReason(t *testing.T) {
	router, store, _, _ := newJobsRouter(t)

	if err := store.Create("u2", "broken.mov"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if recorded, err := store.MarkFailed("u2", "render exploded"); err != nil || !recorded {
		t.Fatalf("mark failed: recorded=%v err=%v", recorded, err)
	}

	rec := doGet(t, router, "/jobs/u2")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /jobs/u2 = %d: %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"stage":"failed"`) || !strings.Contains(body, `"reason":"render exploded"`) {
		t.Errorf("failed job must surface its stage and reason: %s", body)
	}
	if strings.Contains(body, "downloadUrl") {
		t.Errorf("failed job must not offer a downloadUrl: %s", body)
	}
}

func TestUnknownJobIDReturns404(t *testing.T) {
	router, _, _, _ := newJobsRouter(t)

	for _, path := range []string{"/jobs/missing", "/jobs/missing/download"} {
		rec := doGet(t, router, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"status":"fail"`) {
			t.Errorf("GET %s body = %s, want a jsend fail", path, rec.Body)
		}
	}
}

func TestDownloadBeforeDoneReturns404(t *testing.T) {
	router, store, outputs, _ := newJobsRouter(t)

	if err := store.Create("u3", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}

	rec := doGet(t, router, "/jobs/u3/download")
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET download for in-flight job = %d, want 404", rec.Code)
	}
	if len(outputs.opened) != 0 {
		t.Errorf("no output should be opened for an in-flight job, opened %v", outputs.opened)
	}
}

func TestListReturnsEveryProcess(t *testing.T) {
	router, store, _, _ := newJobsRouter(t)

	if err := store.Create("u1", "one.mp4"); err != nil {
		t.Fatalf("create u1: %v", err)
	}
	if err := store.Create("u2", "two.mp4"); err != nil {
		t.Fatalf("create u2: %v", err)
	}

	rec := doGet(t, router, "/jobs")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /jobs = %d: %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"id":"u1"`) || !strings.Contains(body, `"id":"u2"`) {
		t.Errorf("GET /jobs must list every process: %s", body)
	}
	if !strings.Contains(body, `"status":"success"`) {
		t.Errorf("GET /jobs body = %s, want a jsend success", body)
	}
}

func TestDeleteRemovesRowAndCleansObjects(t *testing.T) {
	router, store, _, cleaner := newJobsRouter(t)

	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if recorded, err := store.MarkDone("u1", "outputs/u1"); err != nil || !recorded {
		t.Fatalf("mark done: recorded=%v err=%v", recorded, err)
	}

	rec := doDelete(t, router, "/jobs/u1")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /jobs/u1 = %d: %s", rec.Code, rec.Body)
	}

	if get := doGet(t, router, "/jobs/u1"); get.Code != http.StatusNotFound {
		t.Errorf("GET after delete = %d, want 404", get.Code)
	}
	if list := doGet(t, router, "/jobs"); strings.Contains(list.Body.String(), `"id":"u1"`) {
		t.Errorf("deleted job must vanish from the list: %s", list.Body)
	}

	// both the Upload and the Output objects must be queued for cleanup
	want := map[string]bool{"uploads/u1": false, "outputs/u1": false}
	for _, prefix := range cleaner.deleted {
		if _, ok := want[prefix]; ok {
			want[prefix] = true
		}
	}
	for prefix, cleaned := range want {
		if !cleaned {
			t.Errorf("objects under %s were not cleaned after delete (cleaned %v)", prefix, cleaner.deleted)
		}
	}
}

func TestDeleteInFlightJobIsAllowed(t *testing.T) {
	router, store, _, cleaner := newJobsRouter(t)

	if err := store.Create("u4", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if recorded, err := store.MarkTranscribing("u4"); err != nil || !recorded {
		t.Fatalf("mark transcribing: recorded=%v err=%v", recorded, err)
	}

	rec := doDelete(t, router, "/jobs/u4")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE in-flight /jobs/u4 = %d: %s", rec.Code, rec.Body)
	}
	if get := doGet(t, router, "/jobs/u4"); get.Code != http.StatusNotFound {
		t.Errorf("GET after delete = %d, want 404", get.Code)
	}
	found := false
	for _, prefix := range cleaner.deleted {
		if prefix == "uploads/u4" {
			found = true
		}
	}
	if !found {
		t.Errorf("upload objects were not cleaned after delete (cleaned %v)", cleaner.deleted)
	}
}

func TestDeleteUnknownJobReturns404(t *testing.T) {
	router, _, _, cleaner := newJobsRouter(t)

	rec := doDelete(t, router, "/jobs/missing")
	if rec.Code != http.StatusNotFound {
		t.Errorf("DELETE /jobs/missing = %d, want 404", rec.Code)
	}
	if len(cleaner.deleted) != 0 {
		t.Errorf("unknown job must not touch object storage, cleaned %v", cleaner.deleted)
	}
}
