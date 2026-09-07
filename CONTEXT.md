# Context

Subvision generates subtitles for uploaded videos: an upload is transcribed with
whisper.cpp, then the vfx service renders styled subtitles onto the video with
Remotion and ffmpeg. The Go server orchestrates the pipeline through RabbitMQ;
the Next.js client uploads videos with tus.

## Glossary

- **Upload** — a video a user uploads through the tus endpoint. Stored in object
  storage under the key `uploads/<id>`, where `<id>` is the tus upload id.
- **Transcript** — the ordered sequence of Transcription Segments representing
  the spoken audio of an Upload. Produced during transcription, persisted in the
  job store, and editable by the user before a Re-render.
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
- **Onset** — the start time of a Transcription Segment's first Timed Word;
  the moment a word-driven caption may first become visible. No caption
  mounts before onset, even when the segment window opens earlier
  (ADR-0006); segments without Timed Words are visible from the segment's
  start.
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
  the server's job module; the client polls its status, edits its
  Transcription Segments, and can trigger a Re-render.
- **Re-render** — republishing a fresh VFX Job for a completed Process using
  edited Transcription Segments and the stored original Edit Spec,
  transitioning the Process from done back to rendering. The rendered Output
  replaces `outputs/<id>` in place.
- **Timing Drift** — a Timed Word appearing before it is spoken, or late by a
  fraction of a second. A decode-timing miss, not a missing word or wrong text.

## Invariants

- Object keys are always `<prefix>/<id>`; the id is the last path segment.
- Server and vfx share one bucket (`S3_BUCKET`) for both uploads and outputs;
  the prefix (`uploads/`, `outputs/`) distinguishes them.
- Object storage is accessed only through the S3 API; which S3-compatible store
  runs behind it is an adapter choice (see ADR-0001).
