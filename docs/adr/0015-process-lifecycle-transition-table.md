# ADR-0015: The Process lifecycle is a declared transition table

Date: 2026-11-16 · Status: accepted · Amends ADR-0002

## Context

`internal/job` already guarded the Process lifecycle, but only against terminal
states: a single `UPDATE ... WHERE stage NOT IN ('done','failed')` accepted any
other source stage. The legal graph existed only in the call sites
(`mark`, `Reopen`, `CommitIngestion`) and in ADR-0002's prose. Two consequences:

- An edge could not be reviewed or tested in one place. `uploaded → done`
  passed the guard even though no render had ever been dispatched.
- Changing a stage meant finding every guard by hand. Nothing failed if a call
  site was missed.

Agents reviewing or editing this module had to reconstruct the graph from
scattered `if stage == ...` checks.

## Decision

`server/internal/job/stage.go` declares the lifecycle as one table:

- `Stage` — `uploaded`, `transcribing`, `rendering`, `done`, `failed`.
- `Op` — the intent behind a change: `start-transcription`,
  `commit-ingestion`, `complete`, `fail`, `reopen`.
- `transitions` — one `Transition{From, To, Op}` per legal edge, the only
  declaration of the graph in the codebase.

The store derives its guards from that table:

- `mark(id, op, to, ...)` renders `WHERE id = ? AND stage IN (<sources>)` from
  `Sources(op, to)`, so the edge check and the write are one atomic statement.
  There is no read-then-write window.
- `Reopen` and `CommitIngestion` use the same source list. `CommitIngestion`
  still runs inside its transaction, so a rejected commit persists nothing.
- An unknown destination stage, or an operation with no declared edge, is a
  programming error and returns an error; a legal operation on a Process whose
  current stage is not a declared source reports `recorded=false`, which is the
  existing "unknown or terminal" signal for the queue consumers.

Edges, in full:

```text
uploaded     --start-transcription--> transcribing
transcribing --start-transcription--> transcribing   (redelivered upload job)
transcribing --commit-ingestion-----> rendering
rendering    --complete-------------> done
rendering    --fail-----------------> failed
uploaded     --fail-----------------> failed
transcribing --fail-----------------> failed
done         --reopen---------------> rendering
```

`done` is the only stage with an outgoing edge (`reopen`), and `failed` has
none. Failure stays terminal by construction, which is what
`job_reopen_test.go` already asserted.

## Tests as the verification surface

`internal/job/stage_test.go` holds the machine to account:

- `TestAllowedIsExactlyTheDeclaredMatrix` walks all 5 stages × 5 stages × 5 ops
  and compares `Allowed` against a hand-written copy of the graph. Adding,
  removing, or retargeting an edge fails until both copies agree.
- `TestTransitionsCoverEveryStage` and `TestTransitionsCoverEveryOp` fail when a
  new stage or operation is declared but never wired in.
- `TestOnlyReopenLeavesATerminalStage` — no terminal stage gains an exit by
  accident.
- `TestStoreRefusesUndeclaredEdges` and `TestFailIsAllowedFromEveryInFlightStage`
  prove enforcement against a real SQLite store, not just the pure predicate.

## Alternatives considered

- **Keep terminal-only guards, document the graph in ADR prose.** No failing
  test would catch a wrong call site; rejected.
- **Allow any non-terminal edge per operation (`fail` from anywhere).** That is
  the status quo with more code; it still cannot tell a legal path from a typo.
- **A third-party FSM library.** The machine has five states, one writer, and
  SQL already provides the atomic guard. A dependency buys nothing.
- **Read the current stage, then update.** Rejected: it opens a race between the
  check and the write. The `IN (...)` filter keeps both in one statement.

## Consequences

- The lifecycle is reviewable in one file. A change to it is a change to the
  table plus the test's copy, and the test now fails loudly if they disagree.
- Test fixtures must walk the real path (`createJobWithStage` does:
  uploaded → transcribing → rendering → done). Shortcut setups that jumped
  straight to `done` were rewritten; they were encoding an edge the product
  never takes.
- `Reopen` preserves `output_key`; a re-render keeps the previous Output
  downloadable until a new one lands. `mark` still clears it on non-`done`
  destinations, as before.
- The client's stage list (`client/lib/api.ts`) remains a mirror. It is not
  generated from the table; a stage change still needs both edits, and only the
  server side is test-enforced.
