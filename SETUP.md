# Server Setup

This guide deploys Clave with the supported production stack: Docker Compose plus
a host TLS reverse proxy. PostgreSQL and RabbitMQ stay private; the API and
website bind only to `127.0.0.1`.

Every service reads its configuration from `.env.production`
(`env_file` in `compose.production.yaml`). Each component section below lists the
variables it actually consumes. A variable marked **required** must be set or the
service fails to start.

## Before you start

You need a Linux server with Docker Engine and the Compose plugin, a public DNS
name, and a TLS reverse proxy such as Caddy. Point the name's `A`/`AAAA` record
at the server and allow inbound TCP `80` and `443` only. Do **not** expose
`8000`, `8080`, `5432`, or `5672`.

Deploy a reviewed commit or release tag, not a moving branch:

```bash
sudo install -d -o "$USER" -g "$USER" /opt/clave
git clone https://github.com/cheetahbyte/clave.git /opt/clave/app
cd /opt/clave/app
git checkout <release-tag-or-commit>
```

## Create secrets

Keep the signing key and production environment outside version control:

```bash
install -d -m 700 /opt/clave/secrets
go run ./scripts/generate_jwt_keys.go /opt/clave/secrets/license-jwt-private.pem
cp .env.production.example .env.production
chmod 600 .env.production /opt/clave/secrets/license-jwt-private.pem
```

If Go is unavailable on the server, generate the Ed25519 PKCS#8 key on a trusted
machine and transfer it securely. Back up this key: replacing it invalidates
existing activation-token signatures.

Generate a distinct value for each secret:

```bash
openssl rand -hex 32 # POSTGRES_PASSWORD
openssl rand -hex 32 # RABBITMQ_PASSWORD
openssl rand -hex 32 # LICENSE_HMAC_SECRET
openssl rand -hex 32 # SELF_SERVICE_TOKEN_PEPPER
openssl rand -hex 32 # CSRF_AUTH_KEY
openssl rand -hex 32 # ADMIN_MFA_CODE_PEPPER
openssl rand -hex 32 # WORKER_TOKEN
```

`CSRF_AUTH_KEY` and `ADMIN_MFA_CODE_PEPPER` must each be exactly 64
hexadecimal characters (32 bytes). `WORKER_TOKEN` must be identical for the
backend and the delta worker. Never commit `.env.production` or the signing key.

Build the application services once after checking out the release:

```bash
docker compose --env-file .env.production -f compose.production.yaml build
```

### Compose-level variables

These are read by Compose itself (interpolation and volume mounts), not only by
a container. Keep them URL-safe — hex is a good choice — because two of them are
interpolated into connection URLs.

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `POSTGRES_PASSWORD` | yes | — | Password for the `clave` DB user; interpolated into `DATABASE_URL`. |
| `RABBITMQ_PASSWORD` | yes | — | Password for the `clave` AMQP user; interpolated into `RABBITMQ_URL`. |
| `LICENSE_JWT_PRIVATE_KEY_FILE` | yes | — | Absolute **host** path to the Ed25519 PKCS#8 key. Mounted read-only into the backend at `/run/secrets/license-jwt-private.pem`. |
| `CLAVE_ENV_FILE` | no | `.env.production` | Alternate env file path for `env_file`. |

## Set up PostgreSQL

PostgreSQL stores Clave data. Compose creates the `clave` database and user,
persists data in the `postgres-data` volume, and keeps the service on the private
Compose network.

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `POSTGRES_PASSWORD` | yes | — | Superuser password for the `clave` user. |
| `POSTGRES_USER` | fixed | `clave` | Set by Compose; do not override. |
| `POSTGRES_DB` | fixed | `clave` | Set by Compose; do not override. |

```bash
docker compose --env-file .env.production -f compose.production.yaml up -d postgres
docker compose --env-file .env.production -f compose.production.yaml ps postgres
```

Wait for the service to report `healthy` before starting the backend.

## Set up RabbitMQ

RabbitMQ carries transactional-email and delta-generation jobs. Data persists in
the `rabbitmq-data` volume; the service is private to the Compose network.

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `RABBITMQ_PASSWORD` | yes | — | Password for the default `clave` user (`RABBITMQ_DEFAULT_PASS`). |
| `RABBITMQ_DEFAULT_USER` | fixed | `clave` | Set by Compose; do not override. |

```bash
docker compose --env-file .env.production -f compose.production.yaml up -d rabbitmq
docker compose --env-file .env.production -f compose.production.yaml ps rabbitmq
```

Wait for the service to report `healthy` before starting the backend, emailer,
or delta worker.

## Set up the backend

The backend provides the API, runs migrations, stores update artifacts, and
publishes email events. It depends on healthy PostgreSQL and RabbitMQ.

### Injected by Compose

Set in `compose.production.yaml`; anything you put in `.env.production` for these
names is overridden.

| Variable | Value | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://clave:$POSTGRES_PASSWORD@postgres:5432/clave?sslmode=disable` | PostgreSQL connection string. |
| `RABBITMQ_URL` | `amqp://clave:$RABBITMQ_PASSWORD@rabbitmq:5672/` | AMQP connection string. |
| `LICENSE_JWT_PRIVATE_KEY_FILE` | `/run/secrets/license-jwt-private.pem` | In-container path of the mounted signing key. |
| `MIGRATIONS_DIR` | `/migrations` | Migration files inside the image. |
| `UPDATE_ARTIFACT_STORAGE_PATH` | `/var/lib/clave/update-artifacts` | Backed by the `update-artifacts` volume. |

### Set in `.env.production`

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `LICENSE_HMAC_SECRET` | yes | — | HMAC secret for license keys. Rotating it invalidates existing keys. |
| `SELF_SERVICE_TOKEN_PEPPER` | yes | — | Pepper for hashed self-service tokens. |
| `CSRF_AUTH_KEY` | yes (prod) | — | CSRF signing key; exactly 64 hex characters. |
| `ADMIN_MFA_CODE_PEPPER` | yes (prod) | — | Peppers hashed admin 2FA email codes; exactly 64 hex characters. Legacy name `ADMIN_TOTP_ENCRYPTION_KEY` still works. |
| `WORKER_TOKEN` | yes | — | Shared bearer token the delta worker authenticates with. Must match the worker's value. |
| `PUBLIC_APP_URL` | yes | — | Public base URL, e.g. `https://clave.example.com`. Used in emails and links. |
| `TRUST_PROXY_HEADERS` | recommended | `false` | Honour `X-Forwarded-*`. Safe **only** behind a trusted proxy that overwrites those headers. |
| `RUN_MIGRATIONS` | recommended | `false` | Run DB migrations at startup. Enable for exactly one instance per deployment. |
| `PORT` | no | `8000` | API listen port inside the container. Changing it also requires changing the published port. |
| `DATABASE_MAX_CONNS` | no | `20` | PostgreSQL pool size. Must be greater than zero. |
| `UPDATE_CHECK_RETENTION_DAYS` | no | `90` | Days of update-check history retained; must be 7–365. |
| `CLIENT_CHECKIN_RETENTION_DAYS` | no | `7` | Days of device-linked raw check-ins retained for diagnostics and aggregate retries; must be 1–30. Closed UTC dates are aggregated daily before eligible rows are deleted. |
| `AUDIT_LOG_RETENTION_DAYS` | no | `180` | Days core admin audit events remain identifiable; must be 90–365. |
| `AUDIT_METADATA_RETENTION_DAYS` | no | `90` | Days audit IP addresses and User-Agent strings are retained; must be 30–180 and no greater than audit-log retention. |
| `SELF_SERVICE_RETURN_TOKEN` | no | `false` | Returns self-service tokens in API responses. Debug aid — leave off in production. |
| `DEV` | no | unset | Development mode; relaxes key requirements. **Leave unset in production.** |
| `LOG_LEVEL` | no | `info` | `debug`, `verbose`, or `trace` enable debug logging. |
| `VERBOSE_LOGGING` | no | `false` | Alternative debug-logging switch. |
| `OTEL_ENABLED` | no | `false` | Enable OpenTelemetry metrics. |
| `OTEL_SERVICE_NAME` | no | `clave-api` | Reported service name. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | no | unset | OTLP collector endpoint, e.g. `http://collector:4318`. Required when `OTEL_ENABLED=true`. |

Truthy values are `true`, `1`, and `yes`.

### Client check-in data lifecycle

An authorized client sync updates device-linked current state on the activation:
`last_seen_at`, version, build, platform, architecture, and OS version. Device
hostname and HMAC-protected HWID remain on the registered device for licensing,
support, and self-service deactivation.

A sync that reports a version also creates a raw device-linked check-in without
IP address or User-Agent data. A daily worker aggregates every closed UTC date, normally
making yesterday's adoption counts available the next day. Each activation
counts once per date under its final reported version. Recomputing a date
replaces its aggregate, so retries cannot double-count devices.

Raw check-ins remain for `CLIENT_CHECKIN_RETENTION_DAYS` as a diagnostic and
retry window, then are deleted only after aggregation succeeds. Long-term
`daily_version_adoption` rows retain organization, product, date, version, and
count, but no activation, license, HWID, hostname, or customer identifier.

### Persisted data retention

A centralized worker runs at startup and daily. Cleanup statements are
idempotent, failures are isolated per dataset, and affected row counts are
logged. OpenTelemetry records cleanup outcomes by dataset. Cleanup never runs
on request paths.

| Data | Purpose and sensitivity | Lifecycle |
| --- | --- | --- |
| Self-service tokens | Email, hashed token, expiry, consumption time, creation IP, and optional User-Agent for passwordless access and abuse investigation. | Delete one day after expiry or consumption. |
| Admin sessions | Opaque token and serialized admin session used for authenticated access. | Absolute lifetime 12 hours, idle timeout 30 minutes, immediate deletion on logout, and pgxstore cleanup of expired rows every five minutes. |
| Organization invites | Invitee email, role, hashed token, inviter, expiry, and acceptance time. | Delete accepted invites after 30 days; delete unaccepted invites 30 days after expiry. |
| Admin MFA codes | Admin reference, HMAC code, attempts, expiry, and use time. | Delete one day after expiry or consumption. |
| Admin audit events | Actor, organization, action, resource, timestamp, IP, and User-Agent for accountability and incident investigation. | Scrub IP and User-Agent after `AUDIT_METADATA_RETENTION_DAYS`; delete the core event after `AUDIT_LOG_RETENTION_DAYS`. |
| Update checks | Organization/product/license references and software environment used for update diagnostics. | Delete after `UPDATE_CHECK_RETENTION_DAYS`. |
| Email queue messages | Recipient and transactional payload, which may include MFA codes, links, or license keys. | RabbitMQ removes successful messages immediately. Per-message expiry is 10 minutes for MFA, 15 minutes for self-service links, and seven days for organization invites and license emails. No completed-delivery table is kept. |
| MCP tokens | Hashed organization credential plus creator and last-use metadata. | Retain until regeneration or organization deletion. |
| Delta jobs | Release/artifact checksums and operational status used for idempotency and diagnostics. | Retain with the referenced release/artifact lifecycle. |

Expired and revoked licenses, their devices, and admin accounts are deliberately
excluded from automated retention. Their identity supports licensing, abuse
prevention, customer support, and multi-organization access. They require a
separate product/legal decision before deletion or de-identification.

Ensure `RUN_MIGRATIONS=true` for exactly one backend instance during a
deployment, then start it:

```bash
docker compose --env-file .env.production -f compose.production.yaml up -d backend
docker compose --env-file .env.production -f compose.production.yaml logs --tail=100 backend
```

The API listens only on `127.0.0.1:8000`; verify it locally:

```bash
curl --fail http://127.0.0.1:8000/api/v1/health
```

## Set up the delta worker

The delta worker generates BSDIFF patches for compatible release artifacts. It
requires the backend and RabbitMQ. Full update downloads still work without it.

| Variable | Required | Default | Purpose | Source |
| --- | --- | --- | --- | --- |
| `WORKER_TOKEN` | yes | — | Bearer token used to call the backend. Must match the backend's value. | `.env.production` |
| `API_URL` | yes | `http://localhost:8000` | Backend base URL. | Compose sets `http://backend:8000` |
| `RABBITMQ_URL` | yes | — | AMQP connection string. | Compose |
| `DELTA_MAX_ARTIFACT_BYTES` | no | `67108864` (64 MiB) | Artifacts larger than this are skipped. | Compose |
| `DELTA_MEMORY_BUDGET_BYTES` | no | `1342177280` (1.25 GiB) | BSDIFF memory budget. Must stay below the container's `mem_limit` of 1536 MiB. | Compose |
| `DELTA_HTTP_TIMEOUT_SECONDS` | no | `600` | Timeout for artifact download/upload. | `.env.production` |

```bash
docker compose --env-file .env.production -f compose.production.yaml up -d deltaworker
docker compose --env-file .env.production -f compose.production.yaml logs --tail=100 deltaworker
```

The production service processes one job at a time.

## Set up the website

The website is the static admin and self-service interface. It depends on the
backend and is served only on `127.0.0.1:8080`.

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `VITE_API_BASE_URL` | no | unset | **Build-time** only. Leave unset for the recommended same-origin `/api` proxy. Set it only when the API lives on a different origin, and rebuild the image after changing it. |

The runtime container needs no environment variables.

```bash
docker compose --env-file .env.production -f compose.production.yaml up -d website
curl --fail http://127.0.0.1:8080/healthz
```

## Set up the emailer

The emailer consumes transactional-email events from RabbitMQ and sends them
through SMTP.

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `SMTP_URL` | yes | — | Full SMTP URL, e.g. `smtps://user:pass@smtp.example.com:465` (`smtps://` for implicit TLS, `smtp://` for STARTTLS). URL-encode credentials containing URL metacharacters. |
| `EMAIL_FROM` | no | `Clave <no-reply@clave.dev>` | `From` header. Set it to a domain you control. |
| `RABBITMQ_URL` | yes | `amqp://localhost` | AMQP connection string; injected by Compose. |

The backend's `SMTP_HOST`/`SMTP_PORT`/`SMTP_USER`/`SMTP_PASS`/`MAIL_FROM`
variables are unused in this deployment — the emailer sends all mail.

```bash
docker compose --env-file .env.production -f compose.production.yaml up -d emailer
docker compose --env-file .env.production -f compose.production.yaml logs --tail=100 emailer
```

After the backend creates a license or sends an invite, confirm the emailer log
shows the event was consumed and delivered.

## Configure TLS

Configure Caddy on the host, replacing the hostname:

```caddyfile
clave.example.com {
    handle /api/* {
        reverse_proxy 127.0.0.1:8000
    }

    handle {
        reverse_proxy 127.0.0.1:8080
    }
}
```

Reload Caddy. `TRUST_PROXY_HEADERS=true` is safe only when this trusted proxy
overwrites forwarded headers. The website uses same-origin `/api` requests, so
no frontend API URL is needed.

## Verify and create the first admin

```bash
curl --fail https://clave.example.com/healthz
curl --fail https://clave.example.com/api/v1/health

docker compose --env-file .env.production -f compose.production.yaml exec \
  backend /createadmin admin@example.com 'replace-with-a-strong-password'
```

Sign in at `https://clave.example.com/login`. A six-digit code is published to
RabbitMQ and sent by the emailer on every login, so RabbitMQ, the emailer, and
its SMTP configuration must be working first.

The `createadmin` helper binary needs only `DATABASE_URL`, which Compose
already provides inside the backend container.

## Back up and upgrade

Back up PostgreSQL, the `update-artifacts` volume (when hosting artifacts
locally), `/opt/clave/secrets/license-jwt-private.pem`, and `.env.production`.
Back up RabbitMQ too if queued email must survive a restore.

```bash
git fetch --tags
git checkout <release-tag-or-commit>
docker compose --env-file .env.production -f compose.production.yaml build --pull
docker compose --env-file .env.production -f compose.production.yaml up -d
curl --fail https://clave.example.com/api/v1/health
```

Use `docker compose --env-file .env.production -f compose.production.yaml logs
<service>` to investigate failures. The `references/kepler-infra` directory is
reference infrastructure, not the supported production deployment path.
