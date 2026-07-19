# Clave Performance Pass Design

## Goal

Improve Clave's client startup latency, validation and update-check throughput,
artifact delivery reliability, observability accuracy, and dashboard loading
without breaking the existing client API or requiring new infrastructure.

## Scope

This pass covers the Go API, PostgreSQL queries and migrations, the native
update feed, local and S3-backed artifact delivery, the React administration
website, nginx static delivery, and `CLIENT.md`. Existing activation,
validation, update-check, feed, and download endpoints remain supported.

Redis, a mandatory CDN, a new external queue, and a protocol-v2 replacement
are out of scope. The implementation must remain useful for the supplied
single-host Docker Compose deployment.

## Architecture

### Client synchronization

Add `POST /api/v1/client/sync` as an additive endpoint. It accepts the current
license token, device identifier, and optional update-check fields. It performs
one authorization pass, refreshes the token, returns available channels, and,
when update fields are present, returns an update decision. Existing
`/licenses/validate` and `/updates/check` handlers delegate to shared service
logic and retain their response shapes.

The client guide changes to an offline-first lifecycle: verify the cached JWT
locally, start the application immediately when policy permits, and perform a
deduplicated background sync only when the previous server check is stale or
the token approaches expiry. Clients use exponential backoff with jitter after
transient failures.

### Database hot paths

Create focused queries that load the license and matching active activation in
one round trip. Update configuration queries also return channel requirements,
eliminating the separate feature-gating lookup. Channel metadata is cached
in-process and invalidated by all channel mutations.

Add a partial index matching the latest-published-release lookup order. Rewrite
the admin overview query to aggregate license counts in one scan. Keep indexes
small and aligned with existing query predicates.

PostgreSQL pool configuration is parsed before pool construction and applied
through `pgxpool.NewWithConfig`. `DATABASE_MAX_CONNS` controls pool size and
defaults to 20.

### Update feeds

Retain the existing in-process native-feed cache and batched policy query, then
complete invalidation for every feed-affecting mutation: channel changes,
publish/yank/delete, artifact uploads, changelog edits, and changelog
attachments. Cache entries store both compact JSON and their ETag.

Use single-flight generation per cache key to prevent duplicate work during a
cold miss. Feeds use `Cache-Control: private, no-cache`, allowing client
revalidation without shared-cache storage. `If-None-Match` returns `304`
without rewriting the body. In-process caching is explicitly instance-local and
targets the supplied single-backend deployment. Multi-instance cache
coordination remains out of scope and must be added before horizontally scaling
release-management traffic.

### Update checks and telemetry

Update checks reuse the resolved channel/configuration and avoid repeated token,
license, activation, and channel work. The new sync path shares the same
authorization result with validation.

Recording `update_checks` moves behind a bounded, non-blocking recorder. A full
buffer drops telemetry with a metric rather than delaying client responses.
Shutdown drains the recorder within a bounded timeout. Database insert failures
are logged and counted but do not fail update checks.
`UPDATE_CHECK_RETENTION_DAYS` defaults to 90; the recorder deletes older rows at
startup and every 24 hours. A value of zero disables retention cleanup.

### Artifact delivery

S3-compatible backends continue to return short-lived presigned redirects.
Local files use `http.ServeContent` so clients receive byte ranges,
`Last-Modified`, conditional requests, and resumable downloads. The metrics
response writer preserves optional interfaces needed by optimized HTTP paths.

Artifact upload hashing and storage stream through a temporary file rather than
loading the complete artifact into memory. The temporary file is removed on all
paths. Database metadata is written only after durable storage succeeds, with
best-effort storage cleanup if metadata insertion fails.

The global server no longer imposes a 30-second deadline on long response-body
transfers. Header and request-body protections remain, and upload handlers keep
explicit size limits.

### Observability

Pool metrics use observable gauges or deltas instead of repeatedly adding
cumulative pgx counters. Add query-count or phase-duration measurements around
sync, validation, update checking, feed generation, and telemetry drops.
Metrics remain low-cardinality and never include license, product, or device
identifiers.

### Administration website

Pass the shared `QueryClient` through TanStack Router context. Route guards use
`ensureQueryData`, preventing duplicate current-admin requests. Organization
switches remove organization-scoped caches and fetch only the active route's
data instead of invalidating every known query.

Lazy-load the dashboard chart implementation so the summary renders without
waiting for Recharts. Preserve route code splitting. Enable gzip in nginx and
retain immutable caching for hashed assets.

## Error handling and compatibility

The sync endpoint uses the existing problem response conventions. Omitting
update fields performs validation-only sync. Invalid license state fails the
whole request; an unavailable update configuration returns a successful license
refresh with no update decision and a machine-readable update status. Existing
endpoint status codes and payloads do not change.

Cache failures fall back to feed generation. Telemetry failures never affect
client-facing success. Range requests follow standard HTTP semantics, including
`206`, `416`, and HEAD behavior supplied by `http.ServeContent`.

## Verification

Add unit tests for combined authorization, sync response variants, cache
invalidation, single-flight feed generation, ETag behavior, pool configuration,
metric deltas, telemetry overflow, and update-query reuse. Add HTTP tests for
full, partial, conditional, HEAD, and unsatisfiable artifact requests.

Run Go unit tests, race tests for cache and recorder code, end-to-end tests,
frontend type checking, the production frontend build, and `git diff --check`.
Compare production bundle output and record the main and dashboard chunk sizes.

## Success criteria

- Validation uses one license/activation database round trip plus cached or
  single-query channel resolution.
- Combined launch synchronization avoids duplicate authentication work.
- A cold feed has a bounded query count with no per-release queries; a hot feed
  performs no generation queries.
- Local artifact downloads support resumption and are not capped at 30 seconds.
- Artifact uploads no longer allocate the entire artifact in memory.
- Pool settings actually take effect and exported pool metrics are accurate.
- Initial dashboard authentication performs one current-admin request.
- Static JavaScript and CSS are compressed in production.
- Existing client endpoints and all regression tests continue to pass.
