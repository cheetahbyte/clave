# Client check-in data lifecycle design

## Context

Client sync currently enqueues one `client_checkins` row containing organization, product, license, activation, version, build, platform, architecture, and OS version. A 90-day cleanup job deletes old rows. The version-adoption module reads those rows twice: latest row per activation powers current distribution and the reporting-device table; latest row per activation/day powers the historical chart.

No other product or support path queries `client_checkins`. Device self-service uses `devices` and active `activations`, not check-in history.

## Requirements

- Keep customer name/email and device hostname/HMAC-protected HWID behavior.
- Persist current device state: activation time, last seen, version, build, platform, architecture, and OS version.
- Preserve the version-adoption summary, current distribution, reporting-device table, and daily trend.
- Count one device at most once per day regardless of check-in frequency.
- Keep long-term trends without stable device identifiers.
- Retain device-linked raw check-ins for 7 days, solely as a bounded diagnostic and aggregation source.
- Keep raw rows free of IP addresses and User-Agent strings.
- Make aggregation retry-safe and preserve cascading deletion.

## Discovered behavior

- Producer: `clientsync.Service.Sync` records only successful authorized syncs with a non-empty version.
- Raw writer and cleanup: `diagnostics.Recorder` and `diagnostics.Repository`.
- Current device list and current distribution: `ListLatestClientCheckins`, limited to devices observed during the requested 1–90-day active window.
- Historical chart: `ListDailyLatestClientVersions`, counting distinct activation/day observations by version.
- UI: `website/src/routes/_dash/updates/adoption.tsx` displays both current device state and historical daily counts.
- There is no per-device history view, support query, export, or other consumer.

## Chosen approach

Use **Option B: aggregate before bounded raw data expires**.

This is the smallest change that preserves exact unique-device daily history and the existing 90-day query interface:

1. An authorized version-bearing sync synchronously performs one database operation that:
   - updates current state on the activation;
   - inserts the raw check-in.
2. The recorder retains raw rows for 7 days.
3. Once daily, the aggregation worker recomputes aggregate rows for every closed UTC date still present in raw storage, so yesterday is normally available the next day.
4. It replaces aggregate counts for each recomputed date, then deletes raw rows older than 7 days only when those dates were successfully aggregated. The seven-day window is for diagnostics and retries, not an aggregation delay.
5. Current distribution and the reporting-device table query activation/device current state.
6. Historical trends query daily aggregate rows.

Recomputation uses `INSERT ... ON CONFLICT ... DO UPDATE` plus removal of obsolete dimensions for the same date. Counts are derived from the latest raw row per activation/day, so retries replace counts rather than increment them. Cleanup only runs after successful aggregation.

### Why not continuous counters

Incrementing counters cannot distinguish the first observation from the hundredth without retaining a daily device key. That adds another device-linked dataset and more retry logic. Seven-day raw storage already provides the required deduplication source and short-term diagnostic value.

### Why retain raw rows for seven days

No current feature needs historical device events. Seven days gives operators a bounded recent diagnostic window and gives the daily aggregation job several retry opportunities. Ninety days has no demonstrated functional value.

## Data model

### `activations`

Add nullable current-state columns:

- `last_seen_at timestamptz`
- `current_version text`
- `current_build text`
- `platform text`
- `arch text`
- `os_version text`

Activation time remains `created_at`. Hostname and HMAC-protected HWID remain on `devices`.

Current state is scoped to an activation because activation is the lifecycle currently shown and deactivated by product behavior.

### `client_checkins`

Keep the existing foreign keys to organization, product, license, and activation with cascading deletion. Keep only seven days. Add no IP or User-Agent fields.

### `daily_version_adoption`

Add:

- `date date`
- `organization_id uuid`
- `product_id uuid`
- `version text`
- `device_count bigint`

Primary key: `(date, organization_id, product_id, version)`.

Only version is retained as an analytics dimension. Build, platform, architecture, and OS version remain available in current device state but are excluded from long-term aggregates: the current UI does not chart them, and adding dimensions increases cardinality and re-identification risk without demonstrated value.

This is device-identifier-free aggregated analytics, not anonymous data. The aggregate contains no activation, license, HWID, hostname, or customer identifier. It retains organization/product linkage for tenant-scoped product analytics.

## Active-device definition

For current distribution and the reporting-device table, an active device is an undeactivated activation whose `last_seen_at` is within the caller-selected window (`now() - days`). The existing interface remains 1–90 days and defaults to 30.

For a historical UTC date, a daily active device is an activation that submitted at least one authorized, version-bearing sync on that date. It contributes once, under the version from its final check-in that day.

## Migration

One migration will:

1. add activation current-state columns;
2. create `daily_version_adoption`;
3. backfill current activation state from each activation's newest existing check-in;
4. backfill aggregate rows from the latest check-in per activation/UTC date;
5. leave existing raw rows for the runtime seven-day cleanup to prune after startup.

The down migration drops the aggregate table and new activation columns. Existing raw rows remain compatible with the old implementation.

## Module changes

- Replace the asynchronous `diagnostics.Recorder` write path with one synchronous repository operation that updates activation state and inserts raw history transactionally. Current device state is product data and must not be silently dropped by a full analytics queue.
- Keep the daily aggregation/cleanup worker asynchronous; it is not on the request path.
- Replace raw-history current queries with activation/device queries.
- Replace the daily raw query with an aggregate query.
- Extend recorder cleanup to aggregate closed dates before deletion.
- Set raw retention explicitly to 7 days through configuration, defaulting to 7 and allowing a positive bounded override if operational debugging needs change.
- Preserve the response shape, route, and UI. Update UI wording from “newest check-in” to “current state” where applicable.

## Failure behavior

- A failed state/raw write fails the sync request rather than returning success with stale current device state.
- Aggregation remains off the request path and retries from retained raw rows.
- Aggregation failure prevents deletion, so source rows remain for retry.
- Repeated aggregation replaces the same date/version counts and cannot double-count.

## Validation

- Repository integration test: repeated check-ins for one activation/day yield one aggregate device; the final version wins.
- Repository integration test: repeated aggregation is idempotent.
- Repository integration test: aggregation failure/absence prevents cleanup; successful aggregation permits cleanup.
- Service test: current distribution uses current activation state and daily trend uses aggregate rows.
- Cleanup-worker test: configured retention is passed to cleanup without affecting the sync request path.
- Migration/schema generation: run sqlc generation/check used by the repository.
- Backend targeted tests: `go test ./internal/features/diagnostics ./internal/features/clientsync` from `backend/`.
- Frontend type/build check using the existing website script.

## Excluded scope

- Per-device historical check-in UI or export.
- Aggregate dimensions not consumed by the product.
- IP address or User-Agent collection for client sync.
- Changes to customer identity, hostname, HWID, licensing, or self-service deactivation behavior.
