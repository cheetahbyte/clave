# Clave

Clave is a self-hosted license server. It issues device-bound license and trial
activations as signed JWTs, supports offline validation and update delivery, and
includes an admin dashboard and customer self-service portal.

- **Backend** — Go API and PostgreSQL migrations in `backend/`.
- **Website** — static admin and self-service UI in `website/`.
- **Emailer** — RabbitMQ consumer and SMTP sender in `emailer/`.
- **Client integration** — activation, validation, updates, grace periods, and
  certificate pinning are documented in [`CLIENT.md`](./CLIENT.md).
- **Server deployment** — the supported Docker Compose and TLS-proxy procedure
  is in [`setup.md`](./setup.md).
- **Version adoption** — authorized client synchronization records the running
  version and optional platform diagnostics for the admin dashboard.

## Local development

Generate the required signing key and application secrets:

```bash
go run ./scripts/generate_jwt_keys.go ./license-jwt-private.pem
openssl rand -hex 32 # LICENSE_HMAC_SECRET
openssl rand -hex 32 # SELF_SERVICE_TOKEN_PEPPER
openssl rand -hex 32 # CSRF_AUTH_KEY
openssl rand -hex 32 # ADMIN_MFA_CODE_PEPPER
```

Create `.env`, set `LICENSE_JWT_PRIVATE_KEY_FILE` to the absolute path of the
generated PEM, add the other values, and start the development infrastructure:

```bash
docker compose -f compose.dev.yaml up -d
```

The API is at `http://localhost:8000`, PostgreSQL at `localhost:54321`, and
Mailpit at `http://localhost:8025`. The Compose stack includes the emailer worker,
so forced-development admin 2FA codes arrive in Mailpit. Start the UI with `cd
website && pnpm install && pnpm dev`; Vite proxies `/api` to the backend.

## Production server setup

The production topology is a TLS reverse proxy in front of two loopback-only
ports: the static website on `127.0.0.1:8080` and the API on
`127.0.0.1:8000`. PostgreSQL and RabbitMQ stay on the private Compose network.
The supplied [`compose.production.yaml`](./compose.production.yaml) implements
that layout.

### 1. Prepare the server

Use a supported Linux server with a public DNS name. Point its `A`/`AAAA` record
at the server, allow inbound TCP 80 and 443, and install Docker Engine with the
Compose plugin. Install a TLS reverse proxy such as Caddy on the host. Do not
open ports 8000, 8080, 5432, or 5672 in the firewall.

Create a dedicated location and fetch the repository:

```bash
sudo install -d -o "$USER" -g "$USER" /opt/clave
git clone https://github.com/cheetahbyte/clave.git /opt/clave/app
cd /opt/clave/app
```

For repeatable deployments, check out a release tag or a reviewed commit rather
than deploying a moving branch.

### 2. Generate and store secrets

Create the signing key outside the repository and restrict its permissions:

```bash
install -d -m 700 /opt/clave/secrets
go run ./scripts/generate_jwt_keys.go /opt/clave/secrets/license-jwt-private.pem
cp .env.production.example .env.production
chmod 600 .env.production /opt/clave/secrets/license-jwt-private.pem
```

If Go is not installed on the server, run the generator on a trusted workstation
and securely copy the resulting PEM to that path. The file is an Ed25519 PKCS#8
private key. Back it up: replacing it invalidates the signatures of existing
activation tokens.

Generate independent values and place them in `.env.production`:

```bash
openssl rand -hex 32 # POSTGRES_PASSWORD
openssl rand -hex 32 # RABBITMQ_PASSWORD
openssl rand -hex 32 # LICENSE_HMAC_SECRET
openssl rand -hex 32 # SELF_SERVICE_TOKEN_PEPPER
openssl rand -hex 32 # CSRF_AUTH_KEY (must be exactly 32 bytes / 64 hex chars)
openssl rand -hex 32 # ADMIN_MFA_CODE_PEPPER (same format)
```

Set `PUBLIC_APP_URL` to the public HTTPS origin, configure the emailer's
`SMTP_URL` and `EMAIL_FROM`, and leave `DEV` unset. RabbitMQ is required because
admin 2FA and other transactional email are delivered by the emailer worker. The two Compose passwords should be
URL-safe; the hexadecimal values above are. Never commit `.env.production` or
the PEM.

### 3. Build and start Clave

Build locally on the server, start the dependencies, then start the application:

```bash
docker compose --env-file .env.production -f compose.production.yaml build
docker compose --env-file .env.production -f compose.production.yaml up -d postgres rabbitmq
docker compose --env-file .env.production -f compose.production.yaml up -d backend website emailer
docker compose --env-file .env.production -f compose.production.yaml ps
```

`RUN_MIGRATIONS=true` applies all pending migrations before the API starts. Only
one backend instance should perform migrations during a deployment. The Compose
file persists PostgreSQL, RabbitMQ, and uploaded update artifacts in named
volumes and mounts the signing key read-only.

### 4. Put TLS in front

The backend serves HTTP and must not be exposed directly to the internet. This
Caddy configuration serves the UI and routes same-origin API traffic to the
backend; replace the hostname:

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

Reload Caddy after installing the configuration. Keep
`TRUST_PROXY_HEADERS=true`: Caddy supplies the original scheme, and Clave then
enforces HTTPS, secure cookies, HSTS, redirects plaintext `GET`/`HEAD` requests,
and rejects plaintext requests with bodies. Only enable this setting behind a
trusted proxy that overwrites `X-Forwarded-Proto`, `X-Forwarded-For`, and
`X-Real-IP`.

The website uses a relative `/api` URL by default, so this same-origin layout
needs no frontend build variable and avoids cross-origin cookie problems.

### 5. Verify and create the first admin

Verify both layers before creating an account:

```bash
curl --fail https://clave.example.com/healthz
curl --fail https://clave.example.com/api/v1/health
docker compose --env-file .env.production -f compose.production.yaml logs --tail=100 backend emailer
```

Create the first administrator inside the backend container. Quote the password
so the shell does not interpret punctuation:

```bash
docker compose --env-file .env.production -f compose.production.yaml exec \
  backend /createadmin admin@example.com 'replace-with-a-strong-password'
```

Sign in at `https://clave.example.com/login` and complete two-factor setup.

### 6. Backups, updates, and recovery

Back up all of the following together:

- PostgreSQL (`pg_dump` or a volume/storage snapshot).
- The `update-artifacts` volume if Clave hosts update binaries locally.
- `/opt/clave/secrets/license-jwt-private.pem` and `.env.production` in an
  encrypted secret store.

RabbitMQ contains pending email events; backing it up is optional but prevents
queued messages from being lost. Test restores periodically. A database restore
without the original signing key and application secrets is incomplete.

To update, back up first, check out the desired tag/commit, rebuild, and recreate
the services:

```bash
git fetch --tags
git checkout <release-tag-or-commit>
docker compose --env-file .env.production -f compose.production.yaml build --pull
docker compose --env-file .env.production -f compose.production.yaml up -d
curl --fail https://clave.example.com/api/v1/health
```

Inspect failures with `docker compose --env-file .env.production -f
compose.production.yaml logs <service>`. Admin 2FA codes are published through
RabbitMQ and sent by the emailer, so both services and its SMTP configuration
must be working for anyone to sign in.

## Environment variables

Values described as “truthy” accept `true`, `1`, or `yes` (case-insensitive).
Unless noted otherwise, variables are read at process startup.

### Backend

| Variable | Required/default | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | Default `postgres://clave@localhost:54321/clave?sslmode=disable` | PostgreSQL DSN. Always set an authenticated production value. |
| `DATABASE_MAX_CONNS` | `20` | Maximum PostgreSQL connections used by the API pool. Must be greater than zero. |
| `LICENSE_JWT_PRIVATE_KEY_FILE` | **Required** | Path to exactly one PEM-encoded Ed25519 PKCS#8 private key. |
| `LICENSE_HMAC_SECRET` | **Required** | Secret used when deriving license keys. |
| `SELF_SERVICE_TOKEN_PEPPER` | **Required** | Secret mixed into self-service and organization tokens. |
| `CSRF_AUTH_KEY` | **Required in production** | Exactly 32 random bytes encoded as 64 hex characters; protects CSRF tokens. Dev mode otherwise generates an ephemeral key. |
| `ADMIN_MFA_CODE_PEPPER` | **Required in production** | Exactly 32 random bytes encoded as 64 hex characters; peppers hashed admin 2FA email codes. Dev mode otherwise generates an ephemeral key. The legacy name `ADMIN_TOTP_ENCRYPTION_KEY` is still accepted. |
| `PORT` | `8000` | HTTP listen port. |
| `DEV` | False | Truthy enables insecure development cookies, permits ephemeral CSRF/MFA keys, and skips the emailed 2FA code at login. Never enable in production. |
| `DEV_FORCE_2FA` | False | Only read when `DEV` is truthy. Truthy restores the real emailed 2FA code at login while keeping dev cookies and CSRF handling, so the flow can be tested locally against Mailpit. |
| `RUN_MIGRATIONS` | False | Truthy applies pending Goose migrations before listening. |
| `MIGRATIONS_DIR` | `./migrations` | Migration directory; the backend image uses `/migrations`. |
| `PUBLIC_APP_URL` | Empty | Public website origin used to construct email, portal, invite, and update links. Set the HTTPS production origin without a trailing path. |
| `TRUST_PROXY_HEADERS` | False | Trust proxy IP/scheme headers and enable proxy-aware HTTPS enforcement. Use only behind a trusted overwriting proxy. |
| `SELF_SERVICE_RETURN_TOKEN` | False | Only the exact value `true` returns magic-link tokens in API responses. Development convenience; do not enable in production. |
| `RABBITMQ_URL` | Required when admin 2FA is enabled | AMQP URL. The backend publishes admin 2FA and transactional-email events to `clave.events`; set the same broker on the emailer. |
| `WORKER_TOKEN` | **Required for delta worker** | Shared bearer token protecting `/api/v1/worker/*`; configure the same high-entropy value on the backend and delta worker. |
| `UPDATE_ARTIFACT_STORAGE_PATH` | `./data/update-artifacts` | Persistent local storage for uploaded release artifacts. |
| `UPDATE_CHECK_RETENTION_DAYS` | `90` | Deletes update-check telemetry older than this many days every 24 hours. Set `0` to disable cleanup. |
| `LOG_LEVEL` | `info` | `debug`, `verbose`, or `trace` enables debug logging; other values use info logging. |
| `VERBOSE_LOGGING` | False | Truthy also enables debug logging. |
| `OTEL_ENABLED` | False | Truthy enables OpenTelemetry initialization and export. |
| `OTEL_SERVICE_NAME` | `clave-api` | OpenTelemetry service name. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Empty | OTLP/HTTP collector endpoint, for example `http://collector:4318`. |
| `SMTP_HOST` | Empty | Legacy backend SMTP setting. It is loaded but the current email path does not use it; configure the emailer instead. |
| `SMTP_PORT` | `587` | Legacy backend SMTP port; currently unused. |
| `SMTP_USER` | Empty | Legacy backend SMTP username; currently unused. |
| `SMTP_PASS` | Empty | Legacy backend SMTP password; currently unused. |
| `MAIL_FROM` | `noreply@clave.app` | Legacy backend sender; currently unused. The emailer uses `EMAIL_FROM`. |

Version-adoption check-ins are retained for 90 days. The dashboard counts each
activation once using its latest report from the last 30 days.

### Emailer

| Variable | Required/default | Purpose |
| --- | --- | --- |
| `SMTP_URL` | **Required** | Nodemailer URL: `smtp://user:pass@host:587` or `smtps://...` for implicit TLS. URL-encode credentials. |
| `EMAIL_FROM` | `Clave <no-reply@clave.dev>` | RFC 5322 From value for transactional messages. |
| `RABBITMQ_URL` | `amqp://localhost` | Broker URL. Must point to the broker used by the backend. |

### Website

| Variable | Required/default | Purpose |
| --- | --- | --- |
| `VITE_API_BASE_URL` | Empty | API origin baked into the static bundle at build time. Leave empty for the recommended same-origin `/api` proxy. |

### Delta worker

The Go delta worker consumes durable `delta.generate` events and produces
byte-exact BSDIFF patches between adjacent compatible release artifacts. Release
publication and full downloads continue to work when the worker is unavailable.
The production Compose service runs one job at a time with a 1536 MiB container
limit.

| Variable | Required/default | Purpose |
| --- | --- | --- |
| `API_URL` | `http://localhost:8000` | Backend base URL. |
| `WORKER_TOKEN` | **Required** | Bearer token shared with the backend. |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | Broker containing the durable `clave.delta.generate` queue. |
| `DELTA_MAX_ARTIFACT_BYTES` | `67108864` | Hard ceiling for both source and target artifacts. Oversized jobs are skipped before BSDIFF starts. |
| `DELTA_MEMORY_BUDGET_BYTES` | `1342177280` | Go soft memory limit and input-limit basis. Effective maximum is `min(max artifact bytes, memory budget / 17)`. |
| `DELTA_HTTP_TIMEOUT_SECONDS` | `600` | Timeout for each backend request and artifact transfer. |

Classic BSDIFF generation can use roughly 17 times the source size for suffix
sorting. File-based downloads avoid duplicate transport buffers but do not remove
that algorithmic memory cost. Keep AMQP prefetch and worker concurrency at one;
on a 4 GiB Grid-1 node, start with the documented 64 MiB artifact ceiling and
1536 MiB container limit. Larger artifacts fall back to their full download.

ZIP deltas are useful only when archives are reproducible. Normalize timestamps,
permissions and extra fields, sort entries consistently, and keep the same
DEFLATE settings between releases. For example, build from an already normalized
tree using a stable file list and metadata-stripping mode:

```bash
find app -print | LC_ALL=C sort | zip -X -@ release.zip
strip-nondeterminism release.zip  # when available
```

`zip -X` removes platform-specific extra attributes; it does not normalize file
timestamps by itself. Set stable timestamps in the build tree (commonly via
`SOURCE_DATE_EPOCH`) before creating the archive. Otherwise even a small content
change can move the BSDIFF ratio above 70 percent and Clave records the job as
`skipped`.

### Development, tests, and Compose

| Variable | Required/default | Purpose |
| --- | --- | --- |
| `E2E_BASE_URL` | Test default | Backend base URL used by Go end-to-end tests. |
| `DATABASE_URL` | Test/CLI default | Also used by E2E tests, `createadmin`, Goose, and `psql` recipes. |
| `POSTGRES_PASSWORD` | Production Compose required | Initializes PostgreSQL and interpolates the backend DSN; not read by Clave itself. |
| `RABBITMQ_PASSWORD` | Production Compose required | Initializes RabbitMQ and interpolates service AMQP URLs; not read directly by Clave. |
| `CLAVE_ENV_FILE` | `.env.production` | Optional Compose-only override for the file injected into backend and emailer containers. |

`NODE_ENV=production` is set internally by the emailer image and is not a Clave
configuration input. Standard variables belonging to PostgreSQL, RabbitMQ,
Docker, Go, Bun, pnpm, or Vite are outside this application-specific list.

## Tests

```bash
just test
# or
cd backend && go test ./...
```

## Versioning

Clave uses Semantic Versioning: major releases may change license/API flows,
minor releases add backward-compatible features, and patch releases contain
backward-compatible fixes.
