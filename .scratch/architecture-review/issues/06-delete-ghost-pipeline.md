# 06 — delete-ghost-pipeline

**GitHub issue:** #7

**Status:** closed 2026-09-06 — shipped (GitHub: #7 closed)

## Parent

#1

## What to build

The abandoned synchronous subtitle pipeline is deleted: the commented-out block in the upload processor, the server packages nothing imports, the dead parameter on the tusd registration, and the byte-identical duplicated client hooks. Only the queue-based architecture remains to read. The SRT writer is included — the queue-based path renders via Remotion, and git preserves the file if plain-SRT output ever returns.

## Acceptance criteria

- [x] Unused server packages, the commented-out processor block, and the dead parameter are removed
- [x] Duplicated client hooks are deduplicated
- [x] The server builds and the client typechecks after deletion
- [x] Nothing that remains references the deleted code

## Blocked by

None — can start immediately.
