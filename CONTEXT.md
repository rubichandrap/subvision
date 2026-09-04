# Context

Subvision generates subtitles for uploaded videos: an upload is transcribed with
whisper.cpp, then the vfx service renders styled subtitles onto the video with
Remotion and ffmpeg. The Go server orchestrates the pipeline through RabbitMQ;
the Next.js client uploads videos with tus.

## Glossary

- **Upload** — a video a user uploads through the tus endpoint. Stored in object
  storage under the key `uploads/<id>`, where `<id>` is the tus upload id.
- **Transcription Segment** — one timed subtitle unit produced by whisper:
  start time, end time, text, and the Timed Words inside it. The unit that
  flows from transcription into rendering.
- **Timed Word** — one word of a Transcription Segment with its own start and
  end time, taken from whisper's token timestamps. The timing unit the
  karaoke and pop Animations render against; word timings are never guessed
  from the segment's duration.
- **Caption Page** — a group of Timed Words from one Transcription Segment
  shown together on screen by the karaoke and pop Animations; only the page
  holding the currently spoken word is visible.
- **VFX Job** — the message that tells the vfx service to render a video: the
  upload's object key, its Transcription Segments, and its Edit Spec. Published
  by the server to the vfx queue; consumed by the vfx service. The contract —
  queue names and payload shapes — is defined once per runtime:
  `server/internal/vfxjob` and `vfx/src/contract.ts` mirror each other.
- **Edit Spec** — the creative configuration a user sets in the editor before
  uploading: the trim window, the target Frame, the Animation (resolved to one
  concrete animation by the client), and the Subtitle Style. It travels as
  upload metadata on the tus request, and the VFX Job applies it during render;
  the uploaded file itself is never modified by the client.
- **Frame** — the target canvas of the rendered video: an aspect ratio from a
  Frame Preset or freely dragged by the user, with the source video scaled and
  panned to fill it (crop-to-fill; overflow is discarded).
- **Frame Preset** — a named aspect ratio offered in the editor (9:16, 4:5,
  1:1, 16:9); "Free" means the user drags the frame to any ratio.
- **Subtitle Style** — the visual styling of the rendered subtitles (font,
  size, color, outline, vertical position, and the like), chosen in the editor
  and carried inside the Edit Spec.
- **Output** — the rendered video with burned-in subtitles, stored under
  `outputs/<id>` with the same `<id>` as the Upload.
- **Delete** — the one-way removal of a Process: its record and both stored
  objects (the Upload and, if rendered, the Output) are erased. Irreversible —
  no archive, no restore. A Delete never cancels work already in flight; that
  is Cancel, a separate concept not yet built (see ADR-0004).
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
