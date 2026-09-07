package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/editspec"
	"github.com/rubichandrap/subvision/server/internal/job"
	"github.com/rubichandrap/subvision/server/internal/primitives"
	"github.com/rubichandrap/subvision/server/internal/transcriber"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"
)

// JobReader reads the Process lifecycle; implemented by the job store.
type JobReader interface {
	List() ([]job.Process, error)
	Get(id string) (*job.Process, error)
	Segments(id string) (string, error)
	// EditSpec reads the upload's stored original Edit Spec, or empty when
	// the upload carried none.
	EditSpec(id string) (string, error)
}

// JobDeleter removes a Process record entirely; implemented by the job store.
type JobDeleter interface {
	Delete(id string) (bool, error)
}

// JobWriter persists edited Transcription Segments, reopens a done
// Process for re-render, and fails it when the re-render never left;
// implemented by the job store.
type JobWriter interface {
	SaveSegments(id, segmentsJSON string) error
	Reopen(id string) (bool, error)
	MarkFailed(id, reason string) (bool, error)
}

// RerenderPublisher publishes a fresh VFX Job for a re-render; implemented
// by the vfx publisher.
type RerenderPublisher interface {
	Publish(job vfxjob.Job) error
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
// the gallery, segment edits with validation, a re-render action that
// republishes the render job, plus the one write — an immediate,
// best-effort Delete.
func RegisterJobs(r *gin.Engine, jobs JobReader, writer JobWriter, deleter JobDeleter, publisher RerenderPublisher, outputs OutputOpener, cleaner ObjectCleaner) {
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

	// Read segments serves the stored transcript for one process, so any
	// client can fetch it without touching the queue or the renderer. A
	// process with nothing stored yet reads as an empty list.
	r.GET("/jobs/:id/segments", func(c *gin.Context) {
		id := c.Param("id")
		if _, err := jobs.Get(id); err != nil {
			respondWithError(c, id, err)
			return
		}
		stored, err := jobs.Segments(id)
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
		process, err := jobs.Get(id)
		if err != nil {
			respondWithError(c, id, err)
			return
		}
		if process.Stage != job.StageRendering && process.Stage != job.StageDone {
			primitives.JSendFail(c, gin.H{"id": fmt.Sprintf("job %s is not editable in stage %s", id, process.Stage)}, http.StatusConflict)
			return
		}
		var payload struct {
			Segments json.RawMessage `json:"segments"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			primitives.JSendFail(c, gin.H{"segments": "request body must carry a segments array"}, http.StatusBadRequest)
			return
		}
		var segments []transcriber.Segment
		if err := json.Unmarshal(payload.Segments, &segments); err != nil {
			primitives.JSendFail(c, gin.H{"segments": "segments must decode as timed text"}, http.StatusBadRequest)
			return
		}
		if err := validateSegmentTiming(segments); err != nil {
			primitives.JSendFail(c, gin.H{"segments": err.Error()}, http.StatusBadRequest)
			return
		}
		// Edited segments keep their whisper-original word timings at
		// relative offsets, scaled into the edited window. Match by index:
		// a row without a stored counterpart keeps its submitted words.
		if stored, err := jobs.Segments(id); err == nil && len(stored) > 0 {
			var original []transcriber.Segment
			if json.Unmarshal([]byte(stored), &original) == nil {
				for i := range segments {
					if i >= len(original) {
						break
					}
					if segments[i].Start != original[i].Start || segments[i].End != original[i].End {
						segments[i].Words = rescaleWords(original[i], segments[i])
					} else {
						segments[i].Words = original[i].Words
					}
				}
			}
		}
		raw, err := json.Marshal(segments)
		if err != nil {
			log.Printf("[Jobs] Failed to encode segments for job %s: %v", id, err)
			primitives.JSendError(c, "failed to save segments", http.StatusInternalServerError, nil)
			return
		}
		if err := writer.SaveSegments(id, string(raw)); err != nil {
			log.Printf("[Jobs] Failed to save segments for job %s: %v", id, err)
			primitives.JSendError(c, "failed to save segments", http.StatusInternalServerError, nil)
			return
		}
		primitives.JSendSuccess(c, gin.H{"segments": segments})
	})

	// Re-render republishes a fresh VFX Job with the edited segments plus
	// the upload's original Edit Spec, and moves the process back through
	// rendering to done. The Output overwrites outputs/<id> in place.
	r.POST("/jobs/:id/rerender", func(c *gin.Context) {
		id := c.Param("id")
		process, err := jobs.Get(id)
		if err != nil {
			respondWithError(c, id, err)
			return
		}
		if process.Stage != job.StageDone {
			primitives.JSendFail(c, gin.H{"id": fmt.Sprintf("job %s is not re-renderable in stage %s", id, process.Stage)}, http.StatusConflict)
			return
		}
		stored, err := jobs.Segments(id)
		if err != nil {
			log.Printf("[Jobs] Failed to read segments for job %s: %v", id, err)
			primitives.JSendError(c, "failed to read segments", http.StatusInternalServerError, nil)
			return
		}
		var segments []transcriber.Segment
		if len(stored) > 0 {
			if err := json.Unmarshal([]byte(stored), &segments); err != nil {
				log.Printf("[Jobs] Stored segments for job %s do not decode: %v", id, err)
				primitives.JSendError(c, "stored segments are corrupt", http.StatusInternalServerError, nil)
				return
			}
		}
		rawSpec, err := jobs.EditSpec(id)
		if err != nil {
			log.Printf("[Jobs] Failed to read edit spec for job %s: %v", id, err)
			primitives.JSendError(c, "failed to read edit spec", http.StatusInternalServerError, nil)
			return
		}
		spec, err := editspec.Parse(rawSpec)
		if err != nil {
			log.Printf("[Jobs] Stored edit spec for job %s is invalid: %v", id, err)
			primitives.JSendError(c, "stored edit spec is invalid", http.StatusInternalServerError, nil)
			return
		}
		// Reopen first: a publish without a stage move would strand the
		// render with the process still done, and MarkDone would then
		// refuse the completion. A publish failure after reopen marks the
		// job failed with its reason, never stranded.
		reopened, err := writer.Reopen(id)
		if err != nil {
			log.Printf("[Jobs] Failed to reopen job %s: %v", id, err)
			primitives.JSendError(c, "failed to reopen job", http.StatusInternalServerError, nil)
			return
		}
		if !reopened {
			primitives.JSendFail(c, gin.H{"id": fmt.Sprintf("job %s is not re-renderable in stage %s", id, process.Stage)}, http.StatusConflict)
			return
		}
		rerender := vfxjob.Job{
			UploadID:  id,
			ObjectKey: config.ObjectPrefix + id,
			Segments:  segments,
			EditSpec:  spec,
		}
		if err := publisher.Publish(rerender); err != nil {
			log.Printf("[Jobs] Failed to publish re-render for job %s: %v", id, err)
			if _, markErr := writer.MarkFailed(id, fmt.Sprintf("re-render publish failed: %v", err)); markErr != nil {
				log.Printf("[Jobs] Failed to fail job %s after publish error: %v", id, markErr)
			}
			primitives.JSendError(c, "failed to publish re-render", http.StatusInternalServerError, nil)
			return
		}
		fresh, err := jobs.Get(id)
		if err != nil {
			log.Printf("[Jobs] Failed to read reopened job %s: %v", id, err)
			primitives.JSendError(c, "failed to read reopened job", http.StatusInternalServerError, nil)
			return
		}
		primitives.JSendSuccess(c, newProcessResponse(fresh))
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

// validateSegmentTiming rejects edited segments the render cannot use:
// non-finite times, negative starts, ends at or before their start, and
// segments that start before the previous one ends.
func validateSegmentTiming(segments []transcriber.Segment) error {
	for i, seg := range segments {
		if !finiteTime(seg.Start) || !finiteTime(seg.End) {
			return fmt.Errorf("segment %d must carry finite start and end times", i+1)
		}
		if seg.Start < 0 {
			return fmt.Errorf("segment %d start must not be negative", i+1)
		}
		if seg.End <= seg.Start {
			return fmt.Errorf("segment %d end must be after its start", i+1)
		}
		if i > 0 && seg.Start < segments[i-1].End {
			return fmt.Errorf("segment %d must not start before segment %d ends", i+1, i)
		}
	}
	return nil
}

// rescaleWords keeps a segment's whisper-original word timings at their
// relative offsets, scaled into the edited window. Word timings are never
// re-derived from scratch; a degenerate original window leaves words alone.
func rescaleWords(original, edited transcriber.Segment) []transcriber.Word {
	if len(original.Words) == 0 {
		return original.Words
	}
	span := original.End - original.Start
	if span <= 0 {
		return original.Words
	}
	words := make([]transcriber.Word, len(original.Words))
	for i, w := range original.Words {
		words[i] = transcriber.Word{
			Text:  w.Text,
			Start: edited.Start + (w.Start-original.Start)/span*(edited.End-edited.Start),
			End:   edited.Start + (w.End-original.Start)/span*(edited.End-edited.Start),
		}
	}
	return words
}

func finiteTime(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
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
