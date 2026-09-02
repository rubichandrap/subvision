# ADR-0001: RustFS replaces MinIO as object storage

Date: 2026-09-03 · Status: accepted

## Context

MinIO's community edition was effectively de-platformed for self-hosters in
2025: the admin console was stripped from the community release and official
community Docker images/binaries were pulled, pushing self-hosters toward the
paid AIStor product. This repo had MinIO hardcoded in three places — a
minio-go client in the server, a separate aws-sdk-go-v2 client for tusd's
s3store, and the minio JS SDK in the vfx service — all configured through
`MINIO_*` env vars.

## Decision

- Object storage is **RustFS** (`rustfs/rustfs` image), an Apache-2.0
  S3-compatible store. It is an **adapter** behind an S3 seam, not part of any
  module's interface.
- The seam is named after the interface, not the vendor: env vars are
  **`S3_ENDPOINT` / `S3_ACCESS_KEY` / `S3_SECRET_KEY` / `S3_BUCKET`**, with
  `S3_ENDPOINT` as a full URL and path-style requests.
- The server keeps **one S3 client** (aws-sdk-go-v2): the same client backs
  tusd's s3store and the new `internal/storage` module. The minio-go
  dependency is removed, and `internal/minio` became `internal/storage`
  (bucket injected once; `Upload`/`Download` take a context and paths).
- The vfx service uses **`@aws-sdk/client-s3`** (`forcePathStyle: true`); the
  minio JS SDK is removed.
- Bucket creation and the anonymous-download policy run in an aws-cli compose
  sidecar speaking the S3 API directly (RustFS implements `PutBucketPolicy`),
  gated on RustFS's `/health` healthcheck instead of a sleep.

## Alternatives considered

- **Drop-in image swap, keep the MinIO SDKs** — least churn, but keeps a
  de-platformed vendor in every module's interface and keeps the duplicated
  client. Rejected.
- **SeaweedFS / Garage** — credible open-source alternatives, but RustFS is
  the closest architectural analog with Apache-2.0 licensing and active
  MinIO-migration support. Rejected for this repo.

## Consequences

- `MINIO_*` env names are gone; existing `.env` files must be recreated from
  the updated `.env.example` files.
- `go mod tidy` (when a Go toolchain is available) will drop the stale
  minio-go requirements from `go.sum`.
- Swapping S3-compatible stores later, or pointing at real S3, is now an
  adapter change: the compose image plus the endpoint URL — no module edits.
