package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/rubichandrap/subvision/server/internal/job"
	"github.com/rubichandrap/subvision/server/internal/primitives"
	"github.com/rubichandrap/subvision/server/internal/transcript"
)

// ProcessManager is the unified port the HTTP handler depends on: it covers
// the full Process lifecycle plus segment editing and re-render publishing.
// Implemented by job.Store.
type ProcessManager interface {
	List() ([]job.Process, error)
	Get(id string) (*job.Process, error)
	Segments(id string) (string, error)
	SaveSegments(id string, segments []transcript.Segment) ([]transcript.Segment, error)
	Rerender(id string) (*job.Process, error)
	Delete(id string) (bool, error)
}

// OutputOpener streams an Output object from storage.
type OutputOpener interface {
	Open(ctx context.Context, key string) (io.ReadCloser, int64, error)
}

type processResponse struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Stage       string `json:"stage"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	DownloadURL string `json:"downloadUrl,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

func newProcessResponse(p *job.Process) processResponse {
	resp := processResponse{
		ID:        p.ID,
		Filename:  p.Filename,
		Stage:     string(p.Stage),
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if p.Stage == job.StageDone && p.OutputKey != "" {
		resp.DownloadURL = fmt.Sprintf("/jobs/%s/download", p.ID)
	}
	if p.Stage == job.StageFailed {
		resp.Reason = p.Reason
	}
	return resp
}

// RegisterJobs exposes the status API over the Process lifecycle: reads for
// the gallery, segment edits with validation, a re-render action that
// republishes the render job, plus the one write — an immediate,
// best-effort Delete.
func RegisterJobs(r *gin.Engine, manager ProcessManager, outputs OutputOpener) {
	r.GET("/jobs", func(c *gin.Context) {
		processes, err := manager.List()
		if err != nil {
			log.Printf("[Jobs] Failed to list jobs: %v", err)
			primitives.JSendError(c, "failed to list jobs", http.StatusInternalServerError, nil)
			return
		}
		responses := make([]processResponse, 0, len(processes))
		for i := range processes {
			responses = append(responses, newProcessResponse(&processes[i]))
		}
		primitives.JSendSuccess(c, gin.H{"jobs": responses})
	})

	r.GET("/jobs/:id", func(c *gin.Context) {
		id := c.Param("id")
		process, err := manager.Get(id)
		if err != nil {
			respondWithError(c, id, err)
			return
		}
		primitives.JSendSuccess(c, newProcessResponse(process))
	})

	// Read segments serves the stored transcript for one process, so any
	// client can fetch it without touching the queue or the renderer. A
	// process with nothing stored yet reads as an empty list.
	r.GET("/jobs/:id/segments", func(c *gin.Context) {
		id := c.Param("id")
		if _, err := manager.Get(id); err != nil {
			respondWithError(c, id, err)
			return
		}
		stored, err := manager.Segments(id)
		if err != nil {
			log.Printf("[Jobs] Failed to read segments for job %s: %v", id, err)
			primitives.JSendError(c, "failed to read segments", http.StatusInternalServerError, nil)
			return
		}
		segments := json.RawMessage(stored)
		if len(stored) == 0 {
			segments = json.RawMessage("[]")
		}
		primitives.JSendSuccess(c, gin.H{"segments": segments})
	})

	r.PUT("/jobs/:id/segments", func(c *gin.Context) {
		id := c.Param("id")
		var payload struct {
			Segments json.RawMessage `json:"segments"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			primitives.JSendFail(c, gin.H{"segments": "request body must carry a segments array"}, http.StatusBadRequest)
			return
		}
		var segments []transcript.Segment
		if err := json.Unmarshal(payload.Segments, &segments); err != nil {
			primitives.JSendFail(c, gin.H{"segments": "segments must decode as timed text"}, http.StatusBadRequest)
			return
		}
		saved, err := manager.SaveSegments(id, segments)
		if err != nil {
			respondWithError(c, id, err)
			return
		}
		primitives.JSendSuccess(c, gin.H{"segments": saved})
	})

	// Re-render republishes a fresh VFX Job with the edited segments plus
	// the upload's original Edit Spec, and moves the process back through
	// rendering to done. The Output overwrites outputs/<id> in place.
	r.POST("/jobs/:id/rerender", func(c *gin.Context) {
		id := c.Param("id")
		fresh, err := manager.Rerender(id)
		if err != nil {
			respondWithError(c, id, err)
			return
		}
		primitives.JSendSuccess(c, newProcessResponse(fresh))
	})

	r.GET("/jobs/:id/download", func(c *gin.Context) {
		id := c.Param("id")
		process, err := manager.Get(id)
		if err != nil {
			respondWithError(c, id, err)
			return
		}
		if process.Stage != job.StageDone || process.OutputKey == "" {
			primitives.JSendFail(c, gin.H{"id": fmt.Sprintf("job %s has no output yet", id)}, http.StatusNotFound)
			return
		}

		body, size, err := outputs.Open(c.Request.Context(), process.OutputKey)
		if err != nil {
			log.Printf("[Jobs] Failed to open output %s for job %s: %v", process.OutputKey, id, err)
			primitives.JSendError(c, "failed to open output", http.StatusInternalServerError, nil)
			return
		}
		defer body.Close()

		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", downloadFilename(process)))
		if size > 0 {
			c.DataFromReader(http.StatusOK, size, "video/mp4", body, nil)
			return
		}
		c.Header("Content-Type", "video/mp4")
		c.Status(http.StatusOK)
		if _, err := io.Copy(c.Writer, body); err != nil {
			log.Printf("[Jobs] Failed to stream output for job %s: %v", id, err)
		}
	})

	// Deletion is immediate and best-effort (ADR-0004): the row goes first so
	// the process can never reappear, then the Upload and Output objects go
	// best-effort via job.Store.Delete — an object that fails to vanish is
	// logged, not retried; the UI outcome never hangs on storage errors.
	r.DELETE("/jobs/:id", func(c *gin.Context) {
		id := c.Param("id")
		deleted, err := manager.Delete(id)
		if err != nil {
			log.Printf("[Jobs] Failed to delete job %s: %v", id, err)
			primitives.JSendError(c, "failed to delete job", http.StatusInternalServerError, nil)
			return
		}
		if !deleted {
			primitives.JSendFail(c, gin.H{"id": fmt.Sprintf("no job with id %q", id)}, http.StatusNotFound)
			return
		}
		c.Status(http.StatusNoContent)
	})
}

func respondWithError(c *gin.Context, id string, err error) {
	if errors.Is(err, job.ErrNotFound) {
		primitives.JSendFail(c, gin.H{"id": fmt.Sprintf("no job with id %q", id)}, http.StatusNotFound)
		return
	}
	var sc *job.StageConflictError
	if errors.As(err, &sc) {
		primitives.JSendFail(c, gin.H{"id": sc.Error()}, http.StatusConflict)
		return
	}
	var valErr *transcript.ValidationError
	if errors.As(err, &valErr) {
		primitives.JSendFail(c, gin.H{"segments": valErr.Error()}, http.StatusBadRequest)
		return
	}
	log.Printf("[Jobs] Unexpected error for job %s: %v", id, err)
	primitives.JSendError(c, "internal server error", http.StatusInternalServerError, nil)
}

// downloadFilename derives a friendly attachment name from the original
// upload's filename, falling back to the job id.
func downloadFilename(p *job.Process) string {
	base := path.Base(strings.ReplaceAll(p.Filename, "\\", "/"))
	if base == "" || base == "." || base == "/" {
		return p.ID + ".mp4"
	}
	ext := strings.ToLower(path.Ext(base))
	if ext == ".mp4" {
		return base
	}
	return strings.TrimSuffix(base, path.Ext(base)) + ".mp4"
}
