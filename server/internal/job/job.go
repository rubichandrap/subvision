// Package job owns the Process lifecycle: a job's real, server-side state
// moving from uploaded through transcribing and rendering to done or failed.
// The client reads it through the status API; nothing else invents it.
package job

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
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

type Process struct {
	ID        string
	Filename  string
	Stage     Stage
	Reason    string // why the job failed; empty unless failed
	OutputKey string // where the rendered Output lives; empty unless done
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Tracker records lifecycle transitions as the pipeline crosses them.
type Tracker interface {
	MarkTranscribing(uploadID string) error
	MarkRendering(uploadID string) error
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
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
	return &Store{db: db}, nil
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
// recording the failure reason or output key in the same statement. It
// reports whether the transition took effect, so callers can log the ones
// that didn't instead of dropping them silently.
func (s *Store) mark(id string, stage Stage) (bool, error) {
	return s.update(id, stage, "", "")
}

func (s *Store) update(id string, stage Stage, reason, outputKey string) (bool, error) {
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
func (s *Store) MarkTranscribing(uploadID string) error {
	updated, err := s.mark(uploadID, StageTranscribing)
	if err != nil {
		return err
	}
	if !updated {
		return fmt.Errorf("job %s not found or already terminal, not marking %s", uploadID, StageTranscribing)
	}
	return nil
}

// MarkRendering records that the VFX Job was handed to the vfx service.
func (s *Store) MarkRendering(uploadID string) error {
	updated, err := s.mark(uploadID, StageRendering)
	if err != nil {
		return err
	}
	if !updated {
		return fmt.Errorf("job %s not found or already terminal, not marking %s", uploadID, StageRendering)
	}
	return nil
}

// MarkDone records the completed Output of a rendered job.
func (s *Store) MarkDone(uploadID, outputKey string) error {
	updated, err := s.update(uploadID, StageDone, "", outputKey)
	if err != nil {
		return err
	}
	if !updated {
		return fmt.Errorf("job %s not found or already terminal, not marking %s", uploadID, StageDone)
	}
	return nil
}

// MarkFailed records why a job never produced an Output.
func (s *Store) MarkFailed(uploadID, reason string) error {
	updated, err := s.update(uploadID, StageFailed, reason, "")
	if err != nil {
		return err
	}
	if !updated {
		return fmt.Errorf("job %s not found or already terminal, not marking %s", uploadID, StageFailed)
	}
	return nil
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
