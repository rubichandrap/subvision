package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rubichandrap/subvision/server/internal/db"
	"github.com/rubichandrap/subvision/server/internal/job"
	"github.com/rubichandrap/subvision/server/internal/transcript"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"
)

type fakeOutputs struct {
	body    string
	openErr error
	opened  []string
}

func (f *fakeOutputs) Open(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	body, size, _, err := f.OpenRange(ctx, key, "")
	return body, size, err
}

func (f *fakeOutputs) OpenRange(ctx context.Context, key, byteRange string) (io.ReadCloser, int64, string, error) {
	if f.openErr != nil {
		return nil, 0, "", f.openErr
	}
	f.opened = append(f.opened, key)
	total := int64(len(f.body))
	if byteRange == "" {
		return io.NopCloser(strings.NewReader(f.body)), total, "", nil
	}
	start, end, err := parseByteRange(byteRange, total)
	if err != nil {
		return nil, 0, "", err
	}
	chunk := f.body[start : end+1]
	contentRange := fmt.Sprintf("bytes %d-%d/%d", start, end, total)
	return io.NopCloser(strings.NewReader(chunk)), int64(len(chunk)), contentRange, nil
}

func parseByteRange(raw string, totalSize int64) (int64, int64, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "bytes=") {
		return 0, 0, errors.New("invalid range unit")
	}
	spec := strings.TrimPrefix(raw, "bytes=")
	if strings.Contains(spec, ",") {
		return 0, 0, errors.New("multiple ranges not supported")
	}
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		return 0, 0, errors.New("malformed range header")
	}

	if parts[0] == "" {
		suffix, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || suffix <= 0 {
			return 0, 0, errors.New("invalid suffix range")
		}
		if totalSize <= 0 {
			return 0, 0, ErrRangeUnsatisfiable
		}
		if suffix > totalSize {
			suffix = totalSize
		}
		return totalSize - suffix, totalSize - 1, nil
	}

	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 {
		return 0, 0, errors.New("invalid range start")
	}
	if start >= totalSize {
		return 0, 0, ErrRangeUnsatisfiable
	}

	if parts[1] == "" {
		return start, totalSize - 1, nil
	}

	end, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || end < start {
		return 0, 0, errors.New("invalid range end")
	}
	if end >= totalSize {
		end = totalSize - 1
	}
	return start, end, nil
}

type fakeCleaner struct {
	deleted []string
}

type fakePublisher struct {
	err error
}

func (f fakePublisher) Publish(job vfxjob.Job) error {
	return f.err
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

	outputs := &fakeOutputs{body: "video bytes"}
	cleaner := &fakeCleaner{}
	store, err := job.NewStore(database, fakePublisher{}, cleaner)
	if err != nil {
		t.Fatalf("create job store: %v", err)
	}
	router := gin.New()
	RegisterJobs(router, store, outputs)
	return router, store, outputs, cleaner
}

func doGet(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
func doGetWithHeaders(t *testing.T, router *gin.Engine, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
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
	req := httptest.NewRequest(http.MethodPost, path, nil)
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

	if err := store.CommitIngestion(context.Background(), "u1", nil, nil); err != nil {
		t.Fatalf("commit ingestion: %v", err)
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

func TestSegmentsEndpointServesStoredTranscript(t *testing.T) {
	router, store, _, _ := newJobsRouter(t)

	if err := store.Create("u1", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Nothing stored yet: empty list, not an error.
	rec := doGet(t, router, "/jobs/u1/segments")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /jobs/u1/segments = %d: %s", rec.Code, rec.Body)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"segments":[]`) {
		t.Errorf("unstored segments must read as an empty list: %s", body)
	}

	const stored = `[{"start":1.5,"end":2.5,"text":"hello","words":[{"text":"hello","start":1.6,"end":2.4}]}]`
	var segs []transcript.Segment
	if err := json.Unmarshal([]byte(stored), &segs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := store.CommitIngestion(context.Background(), "u1", segs, nil); err != nil {
		t.Fatalf("commit ingestion: %v", err)
	}

	rec = doGet(t, router, "/jobs/u1/segments")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /jobs/u1/segments = %d: %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"success"`) {
		t.Errorf("segments body = %s, want a jsend success", body)
	}
	for _, want := range []string{`"start":1.5`, `"text":"hello"`, `"words":[{"text":"hello","start":1.6,"end":2.4}]`} {
		if !strings.Contains(body, want) {
			t.Errorf("segments body must carry the render job shape (%s): %s", want, body)
		}
	}

	// Segments outlive the queue message: still readable after done.
	if recorded, err := store.MarkDone("u1", "outputs/u1"); err != nil || !recorded {
		t.Fatalf("mark done: recorded=%v err=%v", recorded, err)
	}
	if rec := doGet(t, router, "/jobs/u1/segments"); !strings.Contains(rec.Body.String(), `"text":"hello"`) {
		t.Errorf("segments must survive done: %s", rec.Body)
	}
}

func TestUnknownJobIDReturns404(t *testing.T) {
	router, _, _, _ := newJobsRouter(t)

	for _, path := range []string{"/jobs/missing", "/jobs/missing/download", "/jobs/missing/segments"} {
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

func TestDownloadSupportsRangeRequests(t *testing.T) {
	router, store, _, _ := newJobsRouter(t)

	if err := store.Create("u-range", "video.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if recorded, err := store.MarkDone("u-range", "outputs/u-range"); err != nil || !recorded {
		t.Fatalf("mark done: %v", err)
	}

	// 1. Full request (no Range header) advertises Accept-Ranges: bytes and inline disposition
	full := doGet(t, router, "/jobs/u-range/download")
	if full.Code != http.StatusOK {
		t.Fatalf("GET without range = %d, want 200: %s", full.Code, full.Body)
	}
	if full.Body.String() != "video bytes" {
		t.Errorf("full body = %q, want 'video bytes'", full.Body.String())
	}
	if accept := full.Header().Get("Accept-Ranges"); accept != "bytes" {
		t.Errorf("Accept-Ranges = %q, want 'bytes'", accept)
	}
	if disp := full.Header().Get("Content-Disposition"); !strings.Contains(disp, "inline") {
		t.Errorf("Content-Disposition = %q, want inline for video playback", disp)
	}

	// 2. Explicit download query param gives attachment disposition
	dl := doGet(t, router, "/jobs/u-range/download?download=true")
	if disp := dl.Header().Get("Content-Disposition"); !strings.Contains(disp, "attachment") {
		t.Errorf("Content-Disposition for ?download=true = %q, want attachment", disp)
	}

	// 3. Range request for first 5 bytes (0-4 of 11)
	part1 := doGetWithHeaders(t, router, "/jobs/u-range/download", map[string]string{
		"Range": "bytes=0-4",
	})
	if part1.Code != http.StatusPartialContent {
		t.Fatalf("GET bytes=0-4 = %d, want 206 Partial Content: %s", part1.Code, part1.Body)
	}
	if part1.Body.String() != "video" {
		t.Errorf("part1 body = %q, want 'video'", part1.Body.String())
	}
	if cr := part1.Header().Get("Content-Range"); cr != "bytes 0-4/11" {
		t.Errorf("Content-Range = %q, want 'bytes 0-4/11'", cr)
	}
	if cl := part1.Header().Get("Content-Length"); cl != "5" {
		t.Errorf("Content-Length = %q, want '5'", cl)
	}

	// 4. Range request from offset to end (bytes=6-)
	part2 := doGetWithHeaders(t, router, "/jobs/u-range/download", map[string]string{
		"Range": "bytes=6-",
	})
	if part2.Code != http.StatusPartialContent {
		t.Fatalf("GET bytes=6- = %d, want 206 Partial Content: %s", part2.Code, part2.Body)
	}
	if part2.Body.String() != "bytes" {
		t.Errorf("part2 body = %q, want 'bytes'", part2.Body.String())
	}
	if cr := part2.Header().Get("Content-Range"); cr != "bytes 6-10/11" {
		t.Errorf("Content-Range = %q, want 'bytes 6-10/11'", cr)
	}

	// 5. Unsatisfiable range (bytes=50-100) returns 416
	unsat := doGetWithHeaders(t, router, "/jobs/u-range/download", map[string]string{
		"Range": "bytes=50-100",
	})
	if unsat.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("GET bytes=50-100 = %d, want 416 Range Not Satisfiable: %s", unsat.Code, unsat.Body)
	}
	if cr := unsat.Header().Get("Content-Range"); cr != "bytes */11" {
		t.Errorf("unsatisfiable Content-Range = %q, want 'bytes */11'", cr)
	}
}
