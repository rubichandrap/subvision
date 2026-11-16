package job

// Stage is one state of the Process lifecycle. The client mirrors this set in
// client/lib/api.ts; keep both lists in step.
type Stage string

const (
	StageUploaded     Stage = "uploaded"
	StageTranscribing Stage = "transcribing"
	StageRendering    Stage = "rendering"
	StageDone         Stage = "done"
	StageFailed       Stage = "failed"
)

// Op names a lifecycle operation: the intent behind a stage change.
type Op string

const (
	OpStartTranscription Op = "start-transcription"
	OpCommitIngestion    Op = "commit-ingestion"
	OpComplete           Op = "complete"
	OpFail               Op = "fail"
	OpReopen             Op = "reopen"
)

// Transition is one legal edge of the Process lifecycle.
type Transition struct {
	From Stage
	To   Stage
	Op   Op
}

// transitions is the single source of truth for the lifecycle: every legal
// stage change, the operation that performs it, and nothing else. The store
// builds its "WHERE stage IN (...)" guards from this table, so an edge missing
// here cannot happen at runtime.
var transitions = []Transition{
	// Ingestion marks the upload as being worked on. The self-edge keeps a
	// redelivered upload job from reporting "unknown or terminal".
	{From: StageUploaded, To: StageTranscribing, Op: OpStartTranscription},
	{From: StageTranscribing, To: StageTranscribing, Op: OpStartTranscription},
	// A committed transcript moves the Process into rendering.
	{From: StageTranscribing, To: StageRendering, Op: OpCommitIngestion},
	// The renderer reports either outcome of a render it was given.
	{From: StageRendering, To: StageDone, Op: OpComplete},
	{From: StageRendering, To: StageFailed, Op: OpFail},
	// Ingestion failures can land before a render ever starts.
	{From: StageUploaded, To: StageFailed, Op: OpFail},
	{From: StageTranscribing, To: StageFailed, Op: OpFail},
	// A re-render reopens a finished Process; failed stays failed.
	{From: StageDone, To: StageRendering, Op: OpReopen},
}

// AllStages lists every stage once, so tests can prove the table covers them.
func AllStages() []Stage {
	return []Stage{StageUploaded, StageTranscribing, StageRendering, StageDone, StageFailed}
}

// Transitions returns a copy of the declared lifecycle edges.
func Transitions() []Transition {
	return append([]Transition(nil), transitions...)
}

// Sources returns the stages from which op may move a Process to to. An empty
// result means that edge is not part of the lifecycle.
func Sources(op Op, to Stage) []Stage {
	var from []Stage
	for _, t := range transitions {
		if t.Op == op && t.To == to {
			from = append(from, t.From)
		}
	}
	return from
}

// Allowed reports whether op may move a Process from one stage to another.
func Allowed(from Stage, op Op, to Stage) bool {
	for _, t := range transitions {
		if t.From == from && t.Op == op && t.To == to {
			return true
		}
	}
	return false
}

// InFlight reports whether the stage still precedes a terminal one.
func (s Stage) InFlight() bool {
	return s == StageUploaded || s == StageTranscribing || s == StageRendering
}
