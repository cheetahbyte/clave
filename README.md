# Clave

A small, self-hosted license server. You give it a product, it hands out license
keys, and your app checks them. Activations are tied to a device and you get back
a signed JWT, so most of the time your client validates offline and only phones
home now and then.

It also does trials (activation without a key), a self-service portal so customers
can manage their own devices, and an admin dashboard for everything else.

- **Backend** — Go, talks Postgres. Lives in `backend/`.
- **Website** — the admin + self-service UI (Vite). Lives in `website/`.
- **Client guide** — building an integration? Read [`CLIENT.md`](./CLIENT.md). It
  covers the encrypted `/activate` and `/validate` flow end to end.

## Quick start

You need a few secrets before anything boots. Generate them once:

```bash
go run ./scripts/generate_jwt_keys.go        # LICENSE_JWT_* keypair (ed25519)
openssl rand -hex 32                         # use for CSRF_AUTH_KEY
openssl rand -hex 32                         # use for ADMIN_TOTP_ENCRYPTION_KEY
```

Drop those into a `.env` file at the repo root (see the table below for the rest),
then bring the stack up:

```bash
docker compose -f compose.dev.yaml up -d
```

That gives you:

| Service  | Where                   | What it is                          |
|----------|-------------------------|-------------------------------------|
| backend  | http://localhost:8000   | the license server                  |
| postgres | localhost:54321         | database                            |
| mailpit  | http://localhost:8025   | fake SMTP — every mail lands here   |

Migrations run automatically on boot (`RUN_MIGRATIONS=true` is set in the compose
file). Create your first admin with:

```bash
docker compose -f compose.dev.yaml exec backend /clave   # server
go run ./backend/cmd/createadmin                          # admin user
```

### Frontend

The admin/self-service UI isn't in the compose file on purpose — you usually want
it running live with hot reload. Start it on the host:

```bash
cd website && pnpm install && pnpm dev   # http://localhost:5173
```

Vite proxies `/api` to the backend on `:8000`, which the compose stack publishes,
so it just works.

### Without Docker

Prefer running it raw? There's a [`justfile`](./justfile). `just dev` runs the
backend (via `air`) and frontend together, `just migrate` applies migrations,
`just test` runs the Go tests. You'll still need Postgres and the same `.env`.

## Configuration

Everything is environment variables. Required ones must be set or the server
won't start; the rest have sane defaults.

### Required

| Variable                    | What it's for                                                                 |
|-----------------------------|-------------------------------------------------------------------------------|
| `DATABASE_URL`              | Postgres connection string. Defaults to the local compose DB.                 |
| `LICENSE_JWT_PRIVATE_KEY`   | ed25519 private key (base64) used to sign license tokens.                     |
| `LICENSE_JWT_PUBLIC_KEY`    | Matching ed25519 public key (base64).                                         |
| `LICENSE_HMAC_SECRET`       | Secret used to derive license keys. Pick something long and random.           |
| `SELF_SERVICE_TOKEN_PEPPER` | Extra secret mixed into self-service magic-link tokens.                       |

In **production** these two are also required (in dev an ephemeral key is generated
for you, with a warning):

| Variable                    | What it's for                                                                 |
|-----------------------------|-------------------------------------------------------------------------------|
| `CSRF_AUTH_KEY`             | Hex-encoded 32-byte key for CSRF protection. Sessions break on restart without it. |
| `ADMIN_TOTP_ENCRYPTION_KEY` | Hex-encoded 32-byte key that encrypts admin 2FA secrets at rest.              |

### Optional

| Variable                     | Default               | What it does                                                        |
|------------------------------|-----------------------|--------------------------------------------------------------------|
| `PORT`                       | `8000`                | Port the server listens on.                                        |
| `DEV`                        | unset                 | Truthy = dev mode: insecure cookies, ephemeral keys allowed.       |
| `PUBLIC_APP_URL`             | —                     | Base URL of the website, used in links inside emails.              |
| `RUN_MIGRATIONS`             | unset                 | Truthy = run DB migrations on startup.                             |
| `LOG_LEVEL` / `VERBOSE_LOGGING` | `info`             | `debug`/`verbose`/`trace` (or a truthy `VERBOSE_LOGGING`) = debug logs. |
| `TRUST_PROXY_HEADERS`        | unset                 | Truthy = trust `X-Forwarded-For` / `X-Real-IP` for client IPs **and** `X-Forwarded-Proto` for the in-app HTTPS guard (redirect HTTP→HTTPS, reject plaintext POSTs). Set only behind a trusted proxy that overwrites these headers. |
| `ADMIN_BEARER_TOKEN`         | —                     | Optional static bearer token for admin API access.                 |
| `SELF_SERVICE_RETURN_TOKEN`  | unset                 | Truthy = return the magic-link token in the API response. Dev convenience. |

### Email (optional)

Mail is only sent if `SMTP_HOST` is set. Locally, Mailpit catches everything.

| Variable     | Default              | What it does                          |
|--------------|----------------------|---------------------------------------|
| `SMTP_HOST`  | —                    | SMTP server host. Set this to enable mail. |
| `SMTP_PORT`  | `587`                | SMTP port.                            |
| `SMTP_USER`  | —                    | SMTP username, if your server needs auth. |
| `SMTP_PASS`  | —                    | SMTP password.                        |
| `MAIL_FROM`  | `noreply@clave.app`  | The From address on outgoing mail.    |

### Update checks (optional)

Used by the `/check-update` endpoint to look up product releases on GitHub.

| Variable        | Default | What it does                                          |
|-----------------|---------|-------------------------------------------------------|
| `GITHUB_REPO`   | —       | `owner/repo` to read releases from. Required for update checks. |
| `GITHUB_TOKEN`  | —       | GitHub token to raise the rate limit / read private repos. |

## Production / HTTPS

The server speaks plain HTTP and is meant to run **behind a TLS-terminating reverse
proxy** (Caddy, nginx, Traefik, …). All transport security comes from TLS — there is no
application-layer payload encryption. **Exposing the backend directly on the internet
without TLS in front is unsupported.**

The proxy should redirect HTTP→HTTPS and terminate TLS. When it does, set
`TRUST_PROXY_HEADERS=true` so the app also enforces HTTPS itself: in production it emits
HSTS, redirects plaintext `GET`/`HEAD` to `https://`, and rejects plaintext body requests
with `403`. Only set this behind a proxy that **overwrites** `X-Forwarded-Proto` — the
header is client-spoofable on a directly exposed server.

Clients should talk to the server over HTTPS only and pin the server's TLS certificate
(SPKI pinning) — see [`CLIENT.md`](./CLIENT.md).

## Tests

```bash
just test          # or: cd backend && go test ./...
```
