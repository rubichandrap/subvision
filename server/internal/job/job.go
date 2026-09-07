// Package job owns the Process lifecycle: a job's real, server-side state
// moving from uploaded through transcribing and rendering to done or failed.
// The client reads it through the status API; nothing else invents it.
package job
import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/editspec"
	"github.com/rubichandrap/subvision/server/internal/transcriber"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"
)
type Stage string

const (
	StageUploaded     Stage = "uploaded"
	StageTranscribing Stage = "transcribing"
	StageRendering    Stage = "rendering"
	StageDone         Stage = "done"
	StageFailed       Stage = "failed"
)

// InFlight reports whether the stage still precedes a terminal one.
func (s Stage) InFlight() bool {
	return s == StageUploaded || s == StageTranscribing || s == StageRendering
}

func (s Stage) valid() bool {
	switch s {
	case StageUploaded, StageTranscribing, StageRendering, StageDone, StageFailed:
		return true
	}
	return false
}

// ErrNotFound is returned by Get when no process carries the requested id.
var ErrNotFound = errors.New("job not found")
// StageConflictError is returned when an operation is invalid for the process's current stage.
type StageConflictError struct {
	ID    string
	Stage Stage
	Op    string
}

func (e *StageConflictError) Error() string {
	return fmt.Sprintf("job %s is not %s in stage %s", e.ID, e.Op, e.Stage)
}

// ErrStageConflict is a sentinel error that matches any StageConflictError via errors.Is.
var ErrStageConflict = errors.New("stage conflict")

func (e *StageConflictError) Is(target error) bool {
	return target == ErrStageConflict
}

// VfxPublisher publishes VFX Jobs; implemented by the rabbitmq publisher and faked in tests.
type VfxPublisher interface {
	Publish(job vfxjob.Job) error
}

// ObjectCleaner removes stored objects under a key prefix best-effort; implemented by the storage client and faked in tests.
type ObjectCleaner interface {
	Delete(ctx context.Context, prefix string) error
}


type Process struct {
	ID        string
	Filename  string
	Stage     Stage
	Reason    string // why the job failed; empty unless failed
	OutputKey string // where the rendered Output lives; empty unless done
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Tracker records lifecycle transitions as the pipeline crosses them. It
// reports whether the transition took effect: false means the id is unknown
// or the job already terminal — callers log that loudly but don't retry;
// a non-nil error means the store itself failed and the transition should
// be attempted again.
type Tracker interface {
	MarkTranscribing(uploadID string) (bool, error)
	MarkRendering(uploadID string) (bool, error)
	// Reopen moves a done job back to rendering for a re-render, bypassing
	// the terminal-stage guard that mark enforces for the normal pipeline.
	Reopen(uploadID string) (bool, error)
	// SaveOriginalSegments persists the whisper-original Transcription Segments for
	// an upload as JSON, so they outlive the queue message.
	SaveOriginalSegments(uploadID, segmentsJSON string) error
	// SaveEditSpec persists the upload's original Edit Spec as JSON, so a
	// re-render reuses it. Empty when the upload carried no spec.
	SaveEditSpec(uploadID, specJSON string) error
}

type Store struct {
	db        *sql.DB
	publisher VfxPublisher
	cleaner   ObjectCleaner
}

func NewStore(db *sql.DB, publisher VfxPublisher, cleaner ObjectCleaner) (*Store, error) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id         TEXT PRIMARY KEY,
			filename   TEXT NOT NULL DEFAULT '',
			stage      TEXT NOT NULL,
			reason     TEXT NOT NULL DEFAULT '',
			output_key TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create jobs table: %w", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS job_segments (
			job_id     TEXT PRIMARY KEY,
			segments   TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create job_segments table: %w", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS job_edit_specs (
			job_id     TEXT PRIMARY KEY,
			spec       TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create job_edit_specs table: %w", err)
	}
	return &Store{
		db:        db,
		publisher: publisher,
		cleaner:   cleaner,
	}, nil
}

// Create records a new process in the uploaded stage, called when the upload
// itself completes.
func (s *Store) Create(id, filename string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO jobs (id, filename, stage, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, filename, string(StageUploaded), now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to record job %s: %w", id, err)
	}
	return nil
}

// mark moves a job into stage unless it is already terminal, optionally
// recording the failure reason or output key in the same statement.
func (s *Store) mark(id string, stage Stage, reason, outputKey string) (bool, error) {
	if !stage.valid() {
		return false, fmt.Errorf("invalid stage %q", stage)
	}
	res, err := s.db.Exec(
		`UPDATE jobs SET stage = ?, reason = ?, output_key = ?, updated_at = ?
		 WHERE id = ? AND stage NOT IN (?, ?)`,
		string(stage), reason, outputKey, time.Now().UTC().Format(time.RFC3339), id,
		string(StageDone), string(StageFailed),
	)
	if err != nil {
		return false, fmt.Errorf("failed to mark job %s as %s: %w", id, stage, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to mark job %s as %s: %w", id, stage, err)
	}
	return affected > 0, nil
}

// MarkTranscribing records that the pipeline started working on the upload.
func (s *Store) MarkTranscribing(uploadID string) (bool, error) {
	return s.mark(uploadID, StageTranscribing, "", "")
}

// MarkRendering records that the VFX Job was handed to the vfx service.
func (s *Store) MarkRendering(uploadID string) (bool, error) {
	return s.mark(uploadID, StageRendering, "", "")
}

// Reopen moves a done job back to rendering for a re-render. The normal
// mark refuses terminal stages, so this runs its own statement: only done
// reopens, failed stays failed, unknown ids report false.
func (s *Store) Reopen(uploadID string) (bool, error) {
	res, err := s.db.Exec(
		`UPDATE jobs SET stage = ?, reason = ?, updated_at = ?
		 WHERE id = ? AND stage = ?`,
		string(StageRendering), "", time.Now().UTC().Format(time.RFC3339), uploadID,
		string(StageDone),
	)
	if err != nil {
		return false, fmt.Errorf("failed to reopen job %s: %w", uploadID, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to reopen job %s: %w", uploadID, err)
	}
	return affected > 0, nil
}

// MarkDone records the completed Output of a rendered job.
func (s *Store) MarkDone(uploadID, outputKey string) (bool, error) {
	return s.mark(uploadID, StageDone, "", outputKey)
}

// MarkFailed records why a job never produced an Output.
func (s *Store) MarkFailed(uploadID, reason string) (bool, error) {
	return s.mark(uploadID, StageFailed, reason, "")
}

// Delete removes the process row entirely, with its stored segments and
// edit spec. It
// reports whether a row was deleted; deleting an unknown id is not an error.
func (s *Store) Delete(id string) (bool, error) {
	if _, err := s.db.Exec(`DELETE FROM job_segments WHERE job_id = ?`, id); err != nil {
		return false, fmt.Errorf("failed to delete segments for job %s: %w", id, err)
	}
	if _, err := s.db.Exec(`DELETE FROM job_edit_specs WHERE job_id = ?`, id); err != nil {
		return false, fmt.Errorf("failed to delete edit spec for job %s: %w", id, err)
	}
	res, err := s.db.Exec(`DELETE FROM jobs WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("failed to delete job %s: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to delete job %s: %w", id, err)
	}
	if affected == 0 {
		return false, nil
	}

	if s.cleaner != nil {
		for _, prefix := range []string{config.ObjectPrefix + id, config.OutputPrefix + id} {
			if err := s.cleaner.Delete(context.Background(), prefix); err != nil {
				log.Printf("[Job] Failed to clean objects under %s for deleted job %s: %v", prefix, id, err)
			}
		}
	}
	return true, nil
}

// Rerender verifies the Process is in done stage, loads stored segments and Edit Spec,
// transitions stage to rendering, and publishes a fresh vfxjob.Job. If re-render publish
// fails, it transitions the process to failed with the error reason and returns an error.
func (s *Store) Rerender(id string) (*Process, error) {
	proc, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if proc.Stage != StageDone {
		return nil, &StageConflictError{
			ID:    id,
			Stage: proc.Stage,
			Op:    "re-renderable",
		}
	}

	stored, err := s.Segments(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read stored segments for job %s: %w", id, err)
	}
	var segments []transcriber.Segment
	if len(stored) > 0 {
		if err := json.Unmarshal([]byte(stored), &segments); err != nil {
			return nil, fmt.Errorf("stored segments for job %s are corrupt: %w", id, err)
		}
	}

	rawSpec, err := s.EditSpec(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read edit spec for job %s: %w", id, err)
	}
	spec, err := editspec.Parse(rawSpec)
	if err != nil {
		return nil, fmt.Errorf("stored edit spec for job %s is invalid: %w", id, err)
	}

	if s.publisher == nil {
		return nil, errors.New("vfx publisher not configured")
	}

	reopened, err := s.Reopen(id)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen job %s: %w", id, err)
	}
	if !reopened {
		return nil, &StageConflictError{
			ID:    id,
			Stage: proc.Stage,
			Op:    "re-renderable",
		}
	}

	job := vfxjob.Job{
		UploadID:  id,
		ObjectKey: config.ObjectPrefix + id,
		Segments:  segments,
		EditSpec:  spec,
	}

	if err := s.publisher.Publish(job); err != nil {
		failReason := fmt.Sprintf("re-render publish failed: %v", err)
		if _, markErr := s.MarkFailed(id, failReason); markErr != nil {
			log.Printf("[Job] Failed to mark job %s failed after publish error: %v", id, markErr)
		}
		return nil, fmt.Errorf("failed to publish re-render: %w", err)
	}

	fresh, err := s.Get(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read reopened job %s: %w", id, err)
	}
	return fresh, nil
}

// SaveOriginalSegments stores the whisper-original Transcription Segments for an
// upload as JSON, replacing any earlier copy. It outlives the queue message
// so the transcript stays readable after the job reaches done.
func (s *Store) SaveOriginalSegments(uploadID, segmentsJSON string) error {
	_, err := s.db.Exec(
		`INSERT INTO job_segments (job_id, segments, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(job_id) DO UPDATE SET segments = excluded.segments, updated_at = excluded.updated_at`,
		uploadID, segmentsJSON, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to save segments for job %s: %w", uploadID, err)
	}
	return nil
}

// SaveSegments enforces that the Process is in rendering or done stage, validates timings,
// rescales words against whisper originals, and saves to job_segments.
func (s *Store) SaveSegments(id string, segments []transcriber.Segment) ([]transcriber.Segment, error) {
	proc, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if proc.Stage != StageRendering && proc.Stage != StageDone {
		return nil, &StageConflictError{
			ID:    id,
			Stage: proc.Stage,
			Op:    "editable",
		}
	}
	if err := transcriber.ValidateSegmentTiming(segments); err != nil {
		return nil, err
	}

	// Rescale word offsets against whisper originals if present
	if stored, err := s.Segments(id); err == nil && len(stored) > 0 {
		var original []transcriber.Segment
		if json.Unmarshal([]byte(stored), &original) == nil {
			for i := range segments {
				if i >= len(original) {
					break
				}
				if segments[i].Start != original[i].Start || segments[i].End != original[i].End {
					segments[i].Words = transcriber.RescaleWords(original[i], segments[i])
				} else {
					segments[i].Words = original[i].Words
				}
			}
		}
	}

	raw, err := json.Marshal(segments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal segments for job %s: %w", id, err)
	}

	if err := s.SaveOriginalSegments(id, string(raw)); err != nil {
		return nil, err
	}
	return segments, nil
}


// SaveEditSpec stores the upload's original Edit Spec as raw JSON, so a
// re-render reuses it. An empty raw means "no edit" and clears any stored
// copy.
func (s *Store) SaveEditSpec(uploadID, specJSON string) error {
	_, err := s.db.Exec(
		`INSERT INTO job_edit_specs (job_id, spec, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(job_id) DO UPDATE SET spec = excluded.spec, updated_at = excluded.updated_at`,
		uploadID, specJSON, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to save edit spec for job %s: %w", uploadID, err)
	}
	return nil
}

// EditSpec returns the stored original Edit Spec JSON, or empty when the
// upload carried none. Reading an unknown id is not an error.
func (s *Store) EditSpec(uploadID string) (string, error) {
	var spec string
	err := s.db.QueryRow(`SELECT spec FROM job_edit_specs WHERE job_id = ?`, uploadID).Scan(&spec)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read edit spec for job %s: %w", uploadID, err)
	}
	return spec, nil
}

// Segments returns the stored Transcription Segments JSON for an upload, or
// empty when nothing was stored yet. Reading an unknown id is not an error.
func (s *Store) Segments(uploadID string) (string, error) {
	var segments string
	err := s.db.QueryRow(`SELECT segments FROM job_segments WHERE job_id = ?`, uploadID).Scan(&segments)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read segments for job %s: %w", uploadID, err)
	}
	return segments, nil
}

// Get returns the process with the requested id, or ErrNotFound.
func (s *Store) Get(id string) (*Process, error) {
	row := s.db.QueryRow(
		`SELECT id, filename, stage, reason, output_key, created_at, updated_at FROM jobs WHERE id = ?`, id,
	)
	return scanProcess(row.Scan)
}

// List returns every process, newest first.
func (s *Store) List() ([]Process, error) {
	rows, err := s.db.Query(
		`SELECT id, filename, stage, reason, output_key, created_at, updated_at FROM jobs ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var processes []Process
	for rows.Next() {
		process, err := scanProcess(rows.Scan)
		if err != nil {
			return nil, err
		}
		processes = append(processes, *process)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	return processes, nil
}

type scanner func(dest ...any) error

func scanProcess(scan scanner) (*Process, error) {
	var p Process
	var stage, createdAt, updatedAt string
	if err := scan(&p.ID, &p.Filename, &stage, &p.Reason, &p.OutputKey, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to read job: %w", err)
	}
	p.Stage = Stage(stage)

	var err error
	p.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, err
	}
	p.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func parseTime(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp %q: %w", value, err)
	}
	return t, nil
}
