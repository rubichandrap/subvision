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

	"github.com/gin-gonic/gin"
	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/job"
	"github.com/rubichandrap/subvision/server/internal/primitives"
)

// JobReader reads the Process lifecycle; implemented by the job store.
type JobReader interface {
	List() ([]job.Process, error)
	Get(id string) (*job.Process, error)
}

// JobDeleter removes a Process record entirely; implemented by the job store.
type JobDeleter interface {
	Delete(id string) (bool, error)
}

// OutputOpener streams an Output object from storage.
type OutputOpener interface {
	Open(ctx context.Context, key string) (io.ReadCloser, int64, error)
}

// ObjectCleaner deletes every stored object under a key prefix (the Upload
// and the Output ride under `uploads/` and `outputs/`); implemented by the
// storage client.
type ObjectCleaner interface {
	Delete(ctx context.Context, prefix string) error
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
// the gallery, plus the one write — an immediate, best-effort Delete.
func RegisterJobs(r *gin.Engine, jobs JobReader, deleter JobDeleter, outputs OutputOpener, cleaner ObjectCleaner) {
	r.GET("/jobs", func(c *gin.Context) {
		processes, err := jobs.List()
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
		process, err := jobs.Get(id)
		if err != nil {
			respondWithError(c, id, err)
			return
		}
		primitives.JSendSuccess(c, newProcessResponse(process))
	})

	r.GET("/jobs/:id/download", func(c *gin.Context) {
		id := c.Param("id")
		process, err := jobs.Get(id)
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
	// best-effort — an object that fails to vanish is logged, not retried; the
	// UI outcome never hangs on storage errors.
	r.DELETE("/jobs/:id", func(c *gin.Context) {
		id := c.Param("id")
		deleted, err := deleter.Delete(id)
		if err != nil {
			log.Printf("[Jobs] Failed to delete job %s: %v", id, err)
			primitives.JSendError(c, "failed to delete job", http.StatusInternalServerError, nil)
			return
		}
		if !deleted {
			primitives.JSendFail(c, gin.H{"id": fmt.Sprintf("no job with id %q", id)}, http.StatusNotFound)
			return
		}

		for _, prefix := range []string{config.ObjectPrefix + id, config.OutputPrefix + id} {
			if err := cleaner.Delete(c.Request.Context(), prefix); err != nil {
				log.Printf("[Jobs] Failed to clean objects under %s for deleted job %s: %v", prefix, id, err)
			}
		}
		c.Status(http.StatusNoContent)
	})
}

func respondWithError(c *gin.Context, id string, err error) {
	if errors.Is(err, job.ErrNotFound) {
		primitives.JSendFail(c, gin.H{"id": fmt.Sprintf("no job with id %q", id)}, http.StatusNotFound)
		return
	}
	log.Printf("[Jobs] Failed to read job %s: %v", id, err)
	primitives.JSendError(c, "failed to read job", http.StatusInternalServerError, nil)
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
