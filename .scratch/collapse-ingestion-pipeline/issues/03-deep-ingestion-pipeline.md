Promoted: #59

# 03 — Deep Ingestion Pipeline with Failure Guard and Scratch File Cleanup

**What to build:** An autonomous Ingestion pipeline module that encapsulates the upload-to-render workflow: validates the upload job, coordinates video download, audio extraction, and speech transcription via clean adapters, applies pure timestamp coordinate shifting, commits to the Job store, immediately cleans up temporary working files, and guarantees that any failure transitions the Process to failed automatically.

**Blocked by:** #57 — Pure Transcript Timestamp Shifting, #58 — Atomic Ingestion Commit and Unified VFX Job Publishing.

**Status:** ready-for-agent

- [ ] A new deep module (`internal/ingest`) exposes `ProcessUpload(ctx context.Context, job UploadJob) error`.
- [ ] Accepts clean adapter interfaces: `VideoDownloader`, `AudioExtractor`, `AudioTranscriber`, and `ProcessLifecycle`.
- [ ] Validates the object key and parses raw Edit Spec metadata via `editspec.Parse`.
- [ ] Coordinates video download, windowed audio extraction (via ffmpeg adapter), and speech transcription (via whisper adapter).
- [ ] Shifts decoded segments into absolute source video coordinates using `transcript.Shift` when a trim window is present.
- [ ] Commits segments and Edit Spec to the Job store using `CommitIngestion`.
- [ ] Scratch video and WAV files are removed immediately via deferred cleanup upon completion or failure.
- [ ] Autonomous failure guard: any failure (missing key, invalid Edit Spec, download error, extraction error, transcription error, commit error) automatically marks the Process as failed with the error cause.
- [ ] Unit tests using in-memory fakes verify success paths, trim window shifting, error guard transitions, and scratch file deletion.
