# 06 — delete-ghost-pipeline

**GitHub issue:** #7

**Status:** ready-for-agent

## Parent

#1

## What to build

The abandoned synchronous subtitle pipeline is deleted: the commented-out block in the upload processor, the server packages nothing imports, the dead parameter on the tusd registration, and the byte-identical duplicated client hooks. Only the queue-based architecture remains to read. The SRT writer is included — the queue-based path renders via Remotion, and git preserves the file if plain-SRT output ever returns.

## Acceptance criteria

- [ ] Unused server packages, the commented-out processor block, and the dead parameter are removed
- [ ] Duplicated client hooks are deduplicated
- [ ] The server builds and the client typechecks after deletion
- [ ] Nothing that remains references the deleted code

## Blocked by

None — can start immediately.
