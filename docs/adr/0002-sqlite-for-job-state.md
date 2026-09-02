# ADR-0002: SQLite for the job lifecycle state

Date: 2026-09-03 · Status: accepted

## Context

Ticket 03 (job status module + status API) needed somewhere to keep a job's
Process lifecycle (uploaded → transcribing → rendering → done/failed). Two
candidates surfaced during the architecture review: revive the dormant
`internal/db` package (SQLite, driver already in `go.mod`), or keep in-memory
state and document the trade-off. The render path is asynchronous — a job
spends minutes in flight — so whatever holds the state outlives individual
HTTP requests and queue messages.

## Decision

- Job state is persisted in **SQLite** through the revived `internal/db`
  package, at `data/subvision.db`. The `jobs` table is created idempotently at
  boot; the pool is capped at one connection (SQLite allows a single writer).
- The choice lives in the `internal/job` package: it owns the lifecycle
  (stages, transitions, terminal-state guards) and is the only module that
  touches the `jobs` table. The status API and the event consumers go through
  it; nothing else knows the storage mechanism.
- Transitions are guarded: a terminal stage (`done`/`failed`) is never
  overwritten by a late event, and a transition that lands on an unknown id is
  reported to the log rather than silently ignored.

## Alternatives considered

- **In-memory state (mutex + map)** — simplest, and adequate for a demo. But
  a server restart while jobs were in flight would orphan them in a prior
  stage forever (with no reconciliation path), which is exactly the
  fake-progress behavior this spec exists to kill. Rejected.
- **Postgres/MySQL** — real durability and concurrency, but adds a service to
  the compose stack for a single-writer workload. Overkill today; the job
  module isolates the choice so a swap later stays local. Rejected.

## Consequences

- Server restarts keep job state: in-flight jobs stay queryable, and done
  downloads keep working.
- `data/` is a bind mount in docker-compose, so state survives container
  rebuilds; keep it out of version control.
- The db path is not currently an environment variable — changing it is a
  one-line edit in `main.go`; promote it to config if it ever varies.
