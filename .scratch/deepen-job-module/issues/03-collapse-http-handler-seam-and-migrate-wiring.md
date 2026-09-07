Promoted: #48

## Parent

Part of #45

## What to build

The HTTP transport handler collapses its six fragmented interfaces into two (ProcessManager and OutputOpener), becoming a thin Gin adapter that unmarshals JSON, delegates to the Job module, and maps domain errors to HTTP status codes (400 for validation errors, 404 for missing jobs, 409 for stage conflicts). Dependency wiring in main.go passes the unified job.Store directly. The 277-line HTTP-coupled integration test file is replaced by lightweight route-mapping smoke tests.

## Acceptance criteria

- [ ] `handler.RegisterJobs` accepts only `ProcessManager` and `OutputOpener` interfaces
- [ ] Six shallow interfaces (`JobReader`, `JobWriter`, `JobDeleter`, `RerenderPublisher`, etc.) are removed from `handler/jobs.go`
- [ ] HTTP handler delegates segment saving, re-rendering, and deletion directly to `ProcessManager`
- [ ] Domain errors are mapped cleanly to HTTP status codes (400 for `ValidationError`, 404 for `ErrNotFound`, 409 for `ErrStageConflict`)
- [ ] `server/cmd/subvision/main.go` wires `job.Store` directly without passing the same instance repeatedly to fragmented interfaces
- [ ] `jobs_rerender_test.go` HTTP tests are replaced with thin route-mapping smoke tests; all server tests pass green (`./test.sh`)

## Blocked by

- #47 — Deepen Job module with side-effect ports, segment editing, re-render, and deletion
