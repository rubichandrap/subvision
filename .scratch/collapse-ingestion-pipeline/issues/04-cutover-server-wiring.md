Promoted: #60

# 04 — Cutover Server Wiring and Retire Obsolete Processor

**What to build:** Complete cutover to the deepened Ingestion pipeline: wire the upload consumer in the server entrypoint directly to the pipeline with real adapters, delete the shallow processor package, delete error coordination from the entrypoint, and retire obsolete micro-methods from the Job store.

**Blocked by:** #59 — Deep Ingestion Pipeline with Failure Guard and Scratch File Cleanup.

**Status:** ready-for-agent

- [ ] `server/cmd/subvision/main.go` instantiates `ingest.New` with production adapters (S3 storage, ffmpeg command, whisper transcriber, and `job.Store`).
- [ ] The upload consumer callback forwards `ingest.UploadJob` directly to `pipeline.ProcessUpload` without manual validation or error branching.
- [ ] The `failJob` helper in `main.go` is deleted.
- [ ] The shallow `server/internal/processor` package and its tests are completely deleted.
- [ ] Obsolete micro-methods (`SaveOriginalSegments`, `SaveEditSpec`, `MarkRendering`) are retired from the Job store's public interface, migrating existing test setups to `CommitIngestion`.
- [ ] Zero transitional shims, aliases, or dead code remain in the codebase.
- [ ] Full server test suite (`./test.sh`) passes cleanly.
