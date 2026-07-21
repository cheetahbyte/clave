# emailer

Transactional email worker for Clave. Consumes events via RabbitMQ, renders
email templates with [React Email](https://react.email), and sends them over
SMTP.

## Setup

```bash
bun install
```

Required environment variables:

| Var          | Description                    |
| ------------ | ------------------------------ |
| `SMTP_URL`   | Nodemailer SMTP URL            |
| `EMAIL_FROM` | From address (default: Clave <no-reply@clave.dev>) |
| `RABBITMQ_URL` | RabbitMQ connection URL (default: amqp://localhost) |

## Running

```bash
# Start the email worker
bun start

# Watch mode (restarts on file changes)
bun dev

# Publish a test event
bun publish:test recipient@example.com
```

## Previewing templates

```bash
bun email:dev
```

Opens a local preview server at `http://localhost:3000`. Templates live in
`src/emails/`. Each `.tsx` file with a default export shows up in the sidebar.

Shared components go in `src/emails/_components/` — directories prefixed with
`_` are ignored by the preview server.

## Email types

Events are published to the `clave.events` exchange with routing key matching
the `type` field. The worker consumes them from the `clave.email-worker` queue.

### `license.created`

| Field         | Required | Type    | Description                                        |
| ------------- | -------- | ------- | -------------------------------------------------- |
| `licenseKey`  | yes      | string  | The license key to display                         |
| `productName` | no       | string  | Product name (used in subject, heading, intro)     |
| `portalLink`  | no       | string  | URL for the "Manage your licenses" CTA button      |
| `isTrial`     | no       | boolean | Whether this is a trial key (changes copy + subject) |

**Event shape:**

```json
{
  "type": "license.created",
  "email": "user@example.com",
  "data": {
    "licenseKey": "CLAVE-XXXX-YYYY-ZZZZ",
    "productName": "Acme Pro",
    "portalLink": "https://app.clave.dev/portal",
    "isTrial": false
  }
}
```

**Generated subject:** `Your Clave license key` (or `Your Clave trial key` when `isTrial` is true).

### `admin.2fa_code`

| Field | Required | Type | Description |
| --- | --- | --- | --- |
| `code` | yes | string | Six ASCII digits used to verify an admin sign-in. |
| `ttlMinutes` | yes | integer | Positive expiration time in minutes. |

```json
{
  "type": "admin.2fa_code",
  "email": "admin@example.com",
  "data": { "code": "123456", "ttlMinutes": 10 }
}
```

**Generated subject:** `Your Clave verification code`.

## Adding a new email type

1. Create `src/emails/<name>.tsx` — export a default React component using
   React Email primitives (Html, Body, Text, Button, …).
2. Add a `PreviewProps` static property so the preview server receives sample
   data.
3. Extend `EmailEvent` in `src/templates.ts` with the new type and add the
   switch case that calls `render()` / `toPlainText()`.
