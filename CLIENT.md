# Client Implementation Guide

This is everything your client needs to talk to the license server. If you're building an integration in any language, start here.

---

## Overview

The basic idea is simple: **activate once, then validate every so often**. Activation ties a license key to a specific device and hands you back a signed JWT. Validation just refreshes that JWT so you know the license is still good.

A trial is really just activation without a key. Instead of sending a license key, your client asks the server to spin up a time-limited trial for the device. You get back the exact same token you'd get from a normal activation, so everything that happens afterwards (validation, grace periods, all of it) works the same way.

All requests and responses are plain JSON over **HTTPS**. There's no application-layer payload encryption — transport security comes from TLS, so always use `https://` and never talk to the server over plaintext HTTP. For defense against a rogue CA or an untrusted TLS terminator, pin the server's TLS certificate (see [Security Notes](#security-notes)).

Before shipping, provision the server's Ed25519 **public** key with the client
(for example, embed it in the signed application bundle). Clave does not expose
a JWKS or public-key endpoint. Treat a replacement key as an application update,
retain the previous key during a planned rotation, and never fetch a verification
key from an unauthenticated network response.

---

## Activation

`POST /api/v1/client/licenses/activate`

```json
{
  "licenseKey": "LIC-XXXX-XXXX-XXXX-XXXX",
  "productId": "<product uuid>",
  "deviceId": {
    "hwid": "<hardware fingerprint>",
    "hostname": "my-macbook"
  }
}
```

A couple of things that trip people up:

- `productId` is the product's UUID (a string), not a number.
- `deviceId` here is an object. Watch out: the trial endpoint below calls the same object `device` instead.

**About the hardware fingerprint (`hwid`):** build it from stable bits of the machine like its UUID, serial number, or MAC address. Hash those together so you're sending a fixed-length value rather than raw hardware details. The important part is consistency: the same device has to produce the same HWID every single time.

**You'll get back:**

```json
{
  "activationId": "<uuid>",
  "token": "<jwt>",
  "validUntil": 1234567890,
  "maskedEmail": "u***@e***.com",
  "name": "Ada Lovelace",
  "updateChannels": [
    {
      "name": "stable",
      "isDefault": true,
      "description": "Production releases"
    }
  ]
}
```

Stash `token` and `validUntil` in an encrypted local cache; you'll lean on both during offline grace periods. `maskedEmail` is fine to show the user (something like "licensed to u***@e***.com") - it never contains the full address. When the license has a customer name, `name` is returned here; otherwise the field is omitted. Cache it during activation if the client needs it because validation and synchronization responses do not return it. `updateChannels` lists the update channels this license can use.

After verifying the Ed25519 signature, require the `EdDSA` algorithm and
validate the registered claims (`sub`, `aud`, `nbf`, `iat`, and `exp`). Also
require `product_id` to match the integrated product and `hwid` to match the
local fingerprint. `activation_id` identifies this device activation; `features`
is the local entitlement set; `license_exp`, when present, is the license's
absolute expiry. Never enable a feature solely because an unverified token says
it is available.

---

## Validation

`POST /api/v1/client/licenses/validate`

Verify the cached JWT signature and claims locally on launch so licensing never
blocks the first frame on network latency. Start a single background refresh
when the previous server check is stale or the token is approaching
`validUntil`; a margin of at least an hour (or about 10% of the token TTL)
avoids expiry races. Deduplicate refreshes inside the process so concurrent
features share one request. After transient failures, retry with exponential
backoff and random jitter rather than on every foreground event.

```json
{
  "token": "<jwt from activation or last validation>",
  "deviceId": "<same hwid you used at activation>"
}
```

**You'll get back:**

```json
{
  "token": "<refreshed jwt>",
  "validUntil": 1234567890,
  "updateChannels": [
    {
      "name": "stable",
      "isDefault": true,
      "description": "Production releases"
    }
  ]
}
```

Whatever token comes back, swap it in for your cached one. Also refresh your local list of available update channels from `updateChannels`.

If the cached token has only recently expired (within the same seven-day window as the offline grace period), `/validate` can still accept it and return a fresh token after rechecking the license, device, and activation state. So a missed refresh is recoverable — but don't rely on it. Refresh early, treat the grace window as a safety net, not the schedule.

## Combined synchronization

`POST /api/v1/client/sync` combines token refresh, channel refresh, and an
optional update check. Prefer it for scheduled background work because the
server authorizes the license and activation only once.

```json
{
  "token": "<cached jwt>",
  "deviceId": "<same hwid>",
  "version": "1.4.0",
  "build": "140",
  "platform": "macos",
  "arch": "arm64",
  "channel": "stable",
  "clientId": "<stable rollout id>"
}
```

Omit `version` and the other update fields for validation-only synchronization.
The response always contains `token`, `validUntil`, `updateChannels`, and
`updateStatus`. `updateStatus` is `not_requested`, `ok`, or `unavailable`; an
`ok` response also includes the normal update-check payload in `update`.

When `version` is present and the license is authorized, Clave also records a
best-effort client check-in for the Version Adoption dashboard. This happens
independently of update resolution, so the check-in still counts when no update
source is configured or the update provider is temporarily unavailable. Build,
platform, architecture, and OS version are optional diagnostics. Check-in
recording never changes the synchronization response or failure behavior.

---

## Starting a Trial

`POST /api/v1/client/trials/start`

Reach for this when the user doesn't have a license key and you want to give them a time-limited trial tied to their device. The server creates the trial license, activates it, and hands you back the same payload `/activate` does. So as far as your client is concerned, starting a trial *is* activating.

```json
{
  "productId": "<product uuid>",
  "device": {
    "hwid": "<hardware fingerprint>",
    "hostname": "my-macbook"
  }
}
```

Worth noting:

- No `licenseKey`, no `customerEmail`. The server mints the trial for you.
- The device object is called `device` here, not `deviceId` like on the activation endpoint. The fields inside (`hwid`, `hostname`) are exactly the same.
- Use the same `hwid` you'd use for a real activation, derived the same way. The server hashes it to make sure a device only gets one trial.

**You'll get back** the same thing activation gives you:

```json
{
  "activationId": "<uuid>",
  "token": "<jwt>",
  "validUntil": 1234567890,
  "maskedEmail": "",
  "updateChannels": [
    {
      "name": "stable",
      "isDefault": true
    }
  ]
}
```

`maskedEmail` comes back empty for trials since there's no customer email attached, and `name` is omitted by default. Cache `token` and `validUntil` just like you would after activation, then validate on the same schedule and use the same grace-period rules.

**One trial per device, per product.** If this device already started a trial for this product, you'll get a `409`. Take that to mean "trial already used" and nudge the user to buy a license and activate with a real key. Don't bother retrying.

---

## Deactivating a Device

`POST /api/v1/client/licenses/deactivate`

Call this when the user deliberately signs out, transfers the license, or
retires a machine. It frees the activation capacity for that device. Do not call
it merely because a validation request failed.

```json
{
  "licenseKey": "LIC-XXXX-XXXX-XXXX-XXXX",
  "deviceId": "<same hwid used at activation>"
}
```

A successful response is:

```json
{ "ok": true }
```

Clear the cached token only after a successful response. This endpoint uses a
string `deviceId`, unlike activation's `deviceId` object.

---

## Offline Grace Period

If you can't reach the server, fall back to what's in your local cache:

```text
now < validUntil              -> license is fine, proceed
now < validUntil + gracePeriod -> warn the user, but still proceed
now >= validUntil + gracePeriod -> block, and tell them to get back online
```

Seven days is a sensible grace period. Don't push it past the token's TTL (also seven days), or the grace period stops meaning anything.

---

## Error Handling

| Status | Meaning |
| -------- | --------- |
| 400 | Bad request — malformed JSON, unknown field, missing required field, or invalid value |
| 401 | Token unusable — invalid, malformed, bad signature, or expired beyond the refresh grace window |
| 422 | Field validation failed; inspect `errors` and fix the request |
| 403 | Token parsed, but the license/device is no longer valid — license revoked or expired, HWID mismatch, or activation deactivated |
| 404 | License not found |
| 409 | Activation limit reached, or this device already used its trial |
| 500 | Server error - fall back to the cache |

A `401` means the token itself is no good, so the client can't recover by retrying the same token — fall back to the cache and your grace-period logic, or re-activate if the grace window has passed. A `403` or `404` means the license is genuinely no good, so don't retry those. For 5xx responses or plain network failures, lean on the cached token and your grace-period logic.

---

## Security Notes

- **Always use HTTPS.** Refuse plaintext `http://` endpoints, and treat the base URL and any `downloadUrl` as HTTPS-only. The server expects to run behind TLS.
- **Pin the server's TLS certificate.** Pin the SubjectPublicKeyInfo (SPKI) hash rather than the leaf certificate - that survives certificate renewal as long as the key is reused, and you should pin a backup key too so a planned key rotation doesn't brick your clients. This protects you against a rogue CA or an untrusted TLS terminator. Platform helpers: Apple `NSPinnedDomains` / `URLSession` delegate, Android `network_security_config` / OkHttp `CertificatePinner`, or your HTTP stack's pinning hook.
- **Encrypt your local cache.** Use the OS keychain or whatever secure storage your platform offers. Don't leave the JWT sitting in plaintext.
- **Keep the HWID stable.** If it changes between calls, validation will start failing with a 403.

---

## Checking for Updates

### Available Update Channels

`POST /api/v1/client/updates/channels`

Use this when your client needs to refresh the channel list without activating or validating first. The server returns only channels this license can access; feature-gated channels are hidden unless the license has every required feature.

Native feeds return an `ETag`. Retain it and send `If-None-Match` on the next
feed request; a `304` means the cached feed is still current. Artifact downloads
require `Authorization: Bearer <jwt>`; URLs never contain the token. Local
artifact storage supports HTTP byte ranges. Download into a temporary file,
resume with `Range`, verify the published SHA-256, then rename the completed
file atomically. S3-backed artifacts redirect to short-lived presigned URLs and
should use the same resume and checksum workflow.

```json
{
  "token": "<jwt from activation or validation>"
}
```

**You'll get back:**

```json
{
  "updateChannels": [
    {
      "name": "stable",
      "isDefault": true,
      "description": "Production releases"
    },
    {
      "name": "beta",
      "isDefault": false,
      "description": "Early access releases"
    }
  ]
}
```

Use `name` as the `channel` value in update checks. If you're not giving users a channel picker, use the channel where `isDefault` is `true`, falling back to `stable` if the list is empty.

### Check for an Update

`POST /api/v1/client/updates/check`

Use your license JWT to check if a newer version is available. The server resolves which update backend to use based on the product's configuration.

```json
{
  "version": "1.0.0",
  "token": "<jwt from activation or validation>",
  "platform": "macos",
  "channel": "stable",
  "build": "42",
  "arch": "arm64",
  "osVersion": "15.0",
  "clientId": "<stable device identifier>"
}
```

Only `version` and `token` are required. Defaults when omitted:

- `platform`: `"macos"`
- `channel`: `"stable"`

**You'll get back:**

```json
{
  "currentVersion": "1.0.0",
  "latestVersion": "1.1.0",
  "updateAvailable": true,
  "downloadUrl": "https://releases.example.com/myapp-1.1.0.dmg",
  "kind": "update_available",
  "releaseNotes": "Bug fixes and performance improvements.",
  "changelog": "## 1.1.0\n- Fixed crash on launch",
  "changelogUrl": "https://your-instance/api/v1/updates/releases/<releaseId>/changelog.html",
  "artifacts": [
    {
      "type": "full",
      "url": "https://releases.example.com/myapp-1.1.0-arm64.dmg",
      "arch": "arm64",
      "os": "macos",
      "sizeBytes": 52428800,
      "sha256": "abc123..."
    }
  ],
  "metadata": {}
}
```

`kind` can be `"no_update"`, `"update_available"`, `"mandatory_update"`, or `"error"`. Use this to decide whether to show a dismissible update prompt or a mandatory upgrade screen.

`releaseNotes` is short inline text. `changelog` carries the full changelog body (Markdown) when one is attached to the release; `changelogUrl` points at a rendered HTML version. Both are optional and omitted when absent.

The `clientId` field enables deterministic staged rollouts. Send a stable, unique value (e.g. a hash of the machine ID) so the server can consistently bucket your client for percentage-based releases.

### Delta Updates

Clave may return a `delta` artifact after the matching full artifact. The full
artifact is always present and is the required fallback. A delta is offered only
when its `fromVersion` exactly matches the `version` sent in the update check.

```json
{
  "type": "delta",
  "url": "https://your-instance/api/v1/updates/artifacts/<artifactId>/download",
  "sizeBytes": 821034,
  "sha256": "<patch SHA-256>",
  "signature": "<base64 Ed25519 signature>",
  "arch": "arm64",
  "os": "macos",
  "metadata": {
    "schema": "clave.delta/v1",
    "algorithm": "bsdiff",
    "fromVersion": "1.4.0",
    "toVersion": "1.5.0",
    "baseSha256": "<installed artifact SHA-256>",
    "targetSha256": "<full target artifact SHA-256>",
    "patchSha256": "<patch SHA-256>",
    "targetSize": 14682120
  }
}
```

Before applying a delta:

1. Reject unknown schemas or algorithms and require `fromVersion` to equal the installed version.
2. Canonicalize the fixed `clave.delta/v1` metadata object as UTF-8 JSON with no insignificant whitespace and keys in this lexical order: `algorithm`, `baseSha256`, `fromVersion`, `patchSha256`, `schema`, `targetSha256`, `targetSize`, `toVersion`. Verify `signature` as Ed25519 over `clave.delta/v1`, one zero byte, and those canonical bytes. Use the same Clave public key used for JWT verification.
3. Verify the installed artifact against `baseSha256`.
4. Download the patch with the license token in the `Authorization` header and verify both `sha256` and `patchSha256`.
5. Apply BSDIFF into a new temporary file, then verify `targetSize` and `targetSha256`.
6. Atomically replace the installed artifact only after every verification succeeds.
7. Download and install the full artifact after any delta validation, transfer, application, or verification failure.

Clave patches the exact uploaded bytes, including ZIP container metadata. Produce
ZIP releases deterministically—stable timestamps, entry ordering, permissions,
extra fields, and compression settings—or expect most ZIP patches to exceed the
70-percent threshold and be skipped.

---

### Native Feed (all platforms)

`GET /api/v1/updates/products/<productId>/<platform>/<channel>/feed.json`

A static JSON feed listing every published release for a product/platform/channel, with full artifact and rollout metadata. Use this when you want to drive updates yourself rather than calling `/updates/check` per launch. The `/updates/check` endpoint is the simpler choice for most clients; reach for the feed when you need the whole release list at once.

**Channel gating**: Channels with required features need a license token. Pass it via `Authorization: Bearer <jwt>`. The query-string form (`?token=<jwt>`) is not accepted, since the token would leak into the artifact URLs the feed returns (and from there into proxy, cache, and client logs).

**You'll get back:**

```json
{
  "schema": "clave.native.feed/v1",
  "product": "6f1c2e8a-9b3d-4f7a-8c21-2a5e0d4b7f90",
  "platform": "macos",
  "channel": "stable",
  "generatedAt": "2026-01-01T00:00:00Z",
  "releases": [
    {
      "version": "1.1.0",
      "build": "42",
      "releaseNotes": "Bug fixes and performance improvements.",
      "changelog": "## 1.1.0\n- Fixed crash on launch",
      "changelogUrl": "https://your-instance/api/v1/updates/releases/<releaseId>/changelog.html",
      "publishedAt": "2026-01-01T00:00:00Z",
      "mandatory": false,
      "rolloutPercentage": 100,
      "minimumSystemVersion": "13.0",
      "artifacts": [
        {
          "type": "full",
          "arch": "arm64",
          "os": "macos",
          "url": "https://releases.example.com/myapp-1.1.0-arm64.dmg",
          "sizeBytes": 52428800,
          "sha256": "abc123..."
        }
      ]
    }
  ]
}
```

Releases are ordered newest-first. Honor `mandatory` and `rolloutPercentage` yourself — the feed lists every release, so you decide which one to offer. Pick the artifact whose `arch`/`os` matches the running machine. `product` is the product's UUID.
