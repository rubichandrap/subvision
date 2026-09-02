# Context

Subvision generates subtitles for uploaded videos: an upload is transcribed with
whisper.cpp, then the vfx service renders styled subtitles onto the video with
Remotion and ffmpeg. The Go server orchestrates the pipeline through RabbitMQ;
the Next.js client uploads videos with tus.

## Glossary

- **Upload** — a video a user uploads through the tus endpoint. Stored in object
  storage under the key `uploads/<id>`, where `<id>` is the tus upload id.
- **Transcription Segment** — one timed subtitle unit produced by whisper:
  start time, end time, text. The unit that flows from transcription into
  rendering.
- **VFX Job** — the message that tells the vfx service to render a video: the
  upload's object key plus its Transcription Segments. Published by the server
  to the vfx queue; consumed by the vfx service. The contract — queue names
  and payload shapes — is defined once per runtime:
  `server/internal/vfxjob` and `vfx/src/contract.ts` mirror each other.
- **Output** — the rendered video with burned-in subtitles, stored under
  `outputs/<id>` with the same `<id>` as the Upload.
- **Process** — the client-facing lifecycle of a job (uploaded →
  transcribing → rendering → done/failed). Real server-side state, owned by
  the server's job module, exposed read-only by the status API
  (`GET /jobs`, `GET /jobs/:id`); the client polls it and never invents state.

## Invariants

- Object keys are always `<prefix>/<id>`; the id is the last path segment.
- Server and vfx share one bucket (`S3_BUCKET`) for both uploads and outputs;
  the prefix (`uploads/`, `outputs/`) distinguishes them.
- Object storage is accessed only through the S3 API; which S3-compatible store
  runs behind it is an adapter choice (see ADR-0001).
