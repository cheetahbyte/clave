# Client Implementation Guide

This is everything your client needs to talk to the license server. If you're building an integration in any language, start here.

---

## Overview

The basic idea is simple: **activate once, then validate every so often**. Activation ties a license key to a specific device and hands you back a signed JWT. Validation just refreshes that JWT so you know the license is still good.

A trial is really just activation without a key. Instead of sending a license key, your client asks the server to spin up a time-limited trial for the device. You get back the exact same token you'd get from a normal activation, so everything that happens afterwards (validation, grace periods, all of it) works the same way.

All requests and responses are plain JSON over **HTTPS**. There's no application-layer payload encryption — transport security comes from TLS, so always use `https://` and never talk to the server over plaintext HTTP. For defense against a rogue CA or an untrusted TLS terminator, pin the server's TLS certificate (see [Security Notes](#security-notes)).

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
  "updateChannels": [
    {
      "name": "stable",
      "isDefault": true,
      "description": "Production releases"
    }
  ]
}
```

Stash `token` and `validUntil` in an encrypted local cache; you'll lean on both during offline grace periods. `maskedEmail` is fine to show the user (something like "licensed to u***@e***.com") - it never contains the full address. `updateChannels` lists the update channels this license can use.

---

## Validation

`POST /api/v1/client/licenses/validate`

Run this on every launch, and again every few hours while the app is open.

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

`maskedEmail` comes back empty for trials since there's no customer email attached. Cache `token` and `validUntil` just like you would after activation, then validate on the same schedule and use the same grace-period rules.

**One trial per device, per product.** If this device already started a trial for this product, you'll get a `409`. Take that to mean "trial already used" and nudge the user to buy a license and activate with a real key. Don't bother retrying.

---

## Offline Grace Period

If you can't reach the server, fall back to what's in your local cache:

```
now < validUntil              -> license is fine, proceed
now < validUntil + gracePeriod -> warn the user, but still proceed
now >= validUntil + gracePeriod -> block, and tell them to get back online
```

Seven days is a sensible grace period. Don't push it past the token's TTL (also seven days), or the grace period stops meaning anything.

---

## Error Handling

| Status | Meaning |
|--------|---------|
| 400 | Bad request — malformed JSON, missing required field, or invalid value |
| 401 | Invalid or expired token |
| 403 | License revoked or expired |
| 404 | License not found |
| 409 | Activation limit reached, or this device already used its trial |
| 500 | Server error - fall back to the cache |

A 403 or 404 means the license is genuinely no good, so don't retry those. For 5xx responses or plain network failures, lean on the cached token and your grace-period logic.

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

The `clientId` field enables deterministic staged rollouts. Send a stable, unique value (e.g. a hash of the machine ID) so the server can consistently bucket your client for percentage-based releases.

---

### Sparkle Appcast (macOS)

Full-file Sparkle appcast XML is available for macOS apps. Point `SUFeedURL` to:

```
https://your-instance/api/v1/updates/products/<productId>/macos/<channel>/appcast.xml
```

**Architecture**: Append `?arch=arm64` or `?arch=x64` to select a specific artifact variant. The feed picks an exact arch match first, then falls back to `universal`. Omitting `arch` defaults to `universal`.

**Channel gating**: Channels with required features need a license token. Pass it via `?token=<jwt>` or `Authorization: Bearer <jwt>`.

**What's included**: Only full-file downloadable artifacts (`dmg`, `zip`, `pkg`). Delta artifacts are not served.

**Example**:
```
SUFeedURL = https://your-instance/api/v1/updates/products/<productId>/macos/stable/appcast.xml?arch=arm64
```

**Ed25519 Signing**: Set `SPARKLE_ED25519_PUBLIC_KEY` and `SPARKLE_ED25519_PRIVATE_KEY` env vars (base64 raw key bytes) on the server. Artifacts uploaded while these are configured will receive a `sparkle:edSignature` in the appcast. Retrieve the public key at `GET /api/v1/updates/sparkle/public-key` and set it as `SUPublicEDKey` in your macOS app's `Info.plist`.

Generate keys via OpenSSL:
```
openssl genpkey -algorithm Ed25519 -out sparkle_private.pem
openssl pkey -in sparkle_private.pem -pubout -out sparkle_public.pem
# Extract raw base64 keys (no headers)
```
