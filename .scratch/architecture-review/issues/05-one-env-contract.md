# 05 — one-env-contract

**GitHub issue:** #6

**Status:** ready-for-agent

## Parent

#1

## What to build

One documented environment contract across the three services: the variable names (`S3_*`, `RABBITMQ_*`, `PORT`, `TMP_DIR`, `CLIENT_URL`, `WHISPER_MODEL_PATH`) live in a single documented place, and each runtime keeps a thin, typed loader that fails loudly at boot when a required variable is missing. No silent credential or URL defaults remain. The S3 rename already shipped with the storage migration; this ticket finishes the contract for the remaining variables.

## Acceptance criteria

- [ ] The full variable contract is documented in one place (README or a dedicated env doc)
- [ ] All three loaders fail loudly on missing required variables with a clear message naming the variable
- [ ] No loader silently falls back to default credentials or URLs
- [ ] All services boot using only documented variables

## Blocked by

None — can start immediately.
