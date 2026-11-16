package job

import (
	"context"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/transcript"
)

// wantEdges is the Process lifecycle written out by hand, one line per legal
// stage change. It duplicates the table in stage.go on purpose: moving the
// machine means editing both, so an edge cannot appear or vanish silently.
func wantEdges() []Transition {
	return []Transition{
		{From: StageUploaded, To: StageTranscribing, Op: OpStartTranscription},
		{From: StageTranscribing, To: StageTranscribing, Op: OpStartTranscription},
		{From: StageTranscribing, To: StageRendering, Op: OpCommitIngestion},
		{From: StageRendering, To: StageDone, Op: OpComplete},
		{From: StageRendering, To: StageFailed, Op: OpFail},
		{From: StageUploaded, To: StageFailed, Op: OpFail},
		{From: StageTranscribing, To: StageFailed, Op: OpFail},
		{From: StageDone, To: StageRendering, Op: OpReopen},
	}
}

func edgeKey(t Transition) Transition { return t }

func TestAllowedIsExactlyTheDeclaredMatrix(t *testing.T) {
	allowed := map[Transition]bool{}
	for _, edge := range wantEdges() {
		allowed[edgeKey(edge)] = true
	}

	for _, from := range AllStages() {
		for _, to := range AllStages() {
			for _, op := range allOps() {
				want := allowed[Transition{From: from, To: to, Op: op}]
				if got := Allowed(from, op, to); got != want {
					t.Errorf("Allowed(%s, %s, %s) = %v, want %v", from, op, to, got, want)
				}
			}
		}
	}
}

func TestTransitionsCoverEveryStage(t *testing.T) {
	seen := map[Stage]bool{}
	for _, edge := range Transitions() {
		seen[edge.From] = true
		seen[edge.To] = true
	}
	for _, stage := range AllStages() {
		if !seen[stage] {
			t.Errorf("stage %q appears in no transition; wire it into the lifecycle or drop it", stage)
		}
	}
}

func TestTransitionsCoverEveryOp(t *testing.T) {
	seen := map[Op]bool{}
	for _, edge := range Transitions() {
		seen[edge.Op] = true
	}
	for _, op := range allOps() {
		if !seen[op] {
			t.Errorf("op %q moves nothing; declare its edge or drop it", op)
		}
	}
}

func TestTransitionsHaveNoDuplicates(t *testing.T) {
	seen := map[Transition]bool{}
	for _, edge := range Transitions() {
		if seen[edge] {
			t.Errorf("duplicate transition %+v", edge)
		}
		seen[edge] = true
	}
}

func TestOnlyReopenLeavesATerminalStage(t *testing.T) {
	for _, edge := range Transitions() {
		if edge.From.InFlight() {
			continue
		}
		if edge.Op != OpReopen || edge.From != StageDone {
			t.Errorf("terminal stage %s must not move under %s (only done reopens)", edge.From, edge.Op)
		}
	}
}

func allOps() []Op {
	return []Op{OpStartTranscription, OpCommitIngestion, OpComplete, OpFail, OpReopen}
}

// TestStoreRefusesUndeclaredEdges proves the table is enforced, not just
// documented: an operation with no edge from the job's current stage leaves
// the row alone and reports "not recorded".
func TestStoreRefusesUndeclaredEdges(t *testing.T) {
	store := newTestStoreWithPorts(t, &fakePublisher{}, &fakeCleaner{})

	if err := store.Create("u-edge", "clip.mp4"); err != nil {
		t.Fatalf("create: %v", err)
	}

	// uploaded -> done is not an edge: the renderer never ran.
	if recorded, err := store.MarkDone("u-edge", "outputs/u-edge"); err != nil || recorded {
		t.Fatalf("done from uploaded: recorded=%v err=%v, want refused", recorded, err)
	}
	assertStage(t, store, "u-edge", StageUploaded)

	// uploaded -> transcribing is.
	if recorded, err := store.MarkTranscribing("u-edge"); err != nil || !recorded {
		t.Fatalf("transcribing from uploaded: recorded=%v err=%v, want recorded", recorded, err)
	}

	// Redelivery of the same upload job re-records the same stage.
	if recorded, err := store.MarkTranscribing("u-edge"); err != nil || !recorded {
		t.Fatalf("transcribing self-edge: recorded=%v err=%v, want recorded", recorded, err)
	}

	// transcribing -> done is not an edge: a render has not been dispatched.
	if recorded, err := store.MarkDone("u-edge", "outputs/u-edge"); err != nil || recorded {
		t.Fatalf("done from transcribing: recorded=%v err=%v, want refused", recorded, err)
	}
	assertStage(t, store, "u-edge", StageTranscribing)

	if err := store.CommitIngestion(context.Background(), "u-edge", []transcript.Segment{}, nil); err != nil {
		t.Fatalf("commit ingestion: %v", err)
	}
	assertStage(t, store, "u-edge", StageRendering)

	if recorded, err := store.MarkDone("u-edge", "outputs/u-edge"); err != nil || !recorded {
		t.Fatalf("done from rendering: recorded=%v err=%v, want recorded", recorded, err)
	}
	assertStage(t, store, "u-edge", StageDone)

	// done only moves through reopen, never straight to another stage.
	if recorded, err := store.MarkFailed("u-edge", "late event"); err != nil || recorded {
		t.Fatalf("failed from done: recorded=%v err=%v, want refused", recorded, err)
	}
	assertStage(t, store, "u-edge", StageDone)
}

// TestFailIsAllowedFromEveryInFlightStage pins the failure edges: any stage
// before a render result can still fail loudly.
func TestFailIsAllowedFromEveryInFlightStage(t *testing.T) {
	for _, stage := range []Stage{StageUploaded, StageTranscribing, StageRendering} {
		t.Run(string(stage), func(t *testing.T) {
			store := newTestStoreWithPorts(t, &fakePublisher{}, &fakeCleaner{})
			id := "u-fail-" + string(stage)
			createJobWithStage(t, store, id, stage)

			if recorded, err := store.MarkFailed(id, "boom"); err != nil || !recorded {
				t.Fatalf("failed from %s: recorded=%v err=%v, want recorded", stage, recorded, err)
			}
			assertStage(t, store, id, StageFailed)
		})
	}
}

func assertStage(t *testing.T, store *Store, id string, want Stage) {
	t.Helper()
	proc, err := store.Get(id)
	if err != nil {
		t.Fatalf("get %s: %v", id, err)
	}
	if proc.Stage != want {
		t.Errorf("stage = %q, want %q", proc.Stage, want)
	}
}
