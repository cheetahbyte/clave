# Persisted data retention design

## Scope

Add explicit lifecycle management for temporary/security data and audit history. Preserve licensing semantics, devices, customer identity, orphaned admin accounts, and the client check-in lifecycle.

## Policies

| Dataset | Policy |
| --- | --- |
| Self-service tokens | Delete one day after expiry or consumption. |
| Admin sessions | Keep pgxstore's built-in five-minute expired-session cleanup. |
| Organization invites | Delete accepted invites after 30 days and unaccepted invites 30 days after expiry. |
| Admin MFA email codes | Delete one day after expiry or consumption. |
| Audit IP/User-Agent | Set to `NULL` after 90 days. |
| Audit core events | Delete after 180 days. |
| Update checks | Delete after 90 days. Reject zero/unbounded configuration. |
| MCP tokens | Retain until regeneration or organization deletion. |
| Delta jobs | Retain with their release/artifact lifecycle. |
| Licenses/devices/admin users | No automated lifecycle change pending product/legal decisions. |
| Client check-ins | Preserve the separately implemented lifecycle. |

Audit metadata retention is configurable from 30–180 days, audit event retention from 90–365 days, and update-check retention from 7–365 days. Metadata retention cannot exceed event retention. Fixed grace periods for short-lived records remain code constants because they are part of the credential/invite lifecycle rather than operator policy.

## Database cleanup module

Add one retention module with a small interface: start a worker with a query adapter and policies, and close it during application shutdown. It runs once at startup and every 24 hours. Each dataset operation is independent, idempotent, and best-effort; one failure does not block later operations or requests. Every result logs the affected row count and records an OpenTelemetry counter labelled by operation and outcome.

Move update-check deletion out of `UpdateCheckRecorder`; it continues recording asynchronously but no longer owns retention. Session and client-check-in cleanup remain specialized because sessions are library-managed and check-ins require aggregation before deletion.

## Email queue expiry

Use per-message RabbitMQ expiration rather than a blanket queue TTL:

- admin MFA code: 10 minutes;
- self-service magic link: 15 minutes;
- organization invite: 7 days, matching invite validity;
- license created/replaced: 7 days, allowing recovery from prolonged SMTP/worker outages while bounding raw license-key and customer-email persistence;
- delta-generation events: no email policy; they retain existing behavior.

Expiration is attached by the backend publisher, keeping queue topology compatible. Expired messages dead-letter through the existing exchange and are not delivered stale. Successful messages are acknowledged and removed; no completed-delivery history is added.

## Documentation and validation

Document stored fields, purpose, sensitivity, retention, deletion mechanism, and exclusions in `SETUP.md`. Add config entries to `.env.production.example`.

Validate configuration bounds, worker independence/idempotent dispatch, email event expiration, generated sqlc output, backend tests, and diagnostics. Database integration tests may require the local Postgres service.
