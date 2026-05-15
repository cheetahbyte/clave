# Client Implementation Guide

This doc covers everything your client needs to do to work with the license server.

---

## Overview

The flow is: **activate once → validate periodically**. Activation binds a license key to a device and returns a signed JWT. Validation refreshes that JWT. Both endpoints use payload encryption — plain JSON over the wire is not accepted.

---

## Payload Encryption (required for /activate and /validate)

All request and response bodies on `/activate` and `/validate` are AES-256-GCM encrypted using a shared secret derived from X25519 ECDH.

**One-time setup:**
1. Fetch the server's static X25519 public key: `GET /api/v1/pubkey`
   ```json
   { "publicKey": "<base64url>" }
   ```
   Pin this key — don't fetch it fresh every time.

**Per request:**
1. Generate an ephemeral X25519 keypair
2. Compute shared secret: `ECDH(your_ephemeral_privkey, server_static_pubkey)`
3. Derive AES key: `HKDF-SHA256(shared_secret, salt=nil, info="clave-v1")` → 32 bytes
4. Encrypt your JSON body with AES-256-GCM:
   - Random 12-byte nonce
   - Wire format: `base64url(nonce || ciphertext)`
5. Send:
   - Header: `X-Client-Public-Key: <base64url of your ephemeral pubkey>`
   - Body: the base64url ciphertext

**Response:**
- Body is also encrypted in the same format — decrypt with the same AES key
- Verify the `X-Client-Key-Echo` response header matches what you sent. If it doesn't, discard the response (replay attack).

---

## Activation

`POST /api/v1/activate` (encrypted)

```json
{
  "licenseKey": "LIC-XXXX-XXXX-XXXX-XXXX",
  "productId": 1,
  "customerEmail": "user@example.com",
  "deviceId": {
    "hwid": "<hardware fingerprint>",
    "hostname": "my-macbook"
  }
}
```

**Hardware fingerprint (`hwid`):** Derive this from stable device identifiers — machine UUID, serial number, MAC address. Hash them together so you're sending a fixed-length string, not raw hardware data. The same device must always produce the same HWID.

**Response:**
```json
{
  "activationId": 42,
  "token": "<jwt>",
  "validUntil": 1234567890
}
```

Store `token` and `validUntil` in an encrypted local cache. You'll need both for offline grace periods.

---

## Validation

`POST /api/v1/validate` (encrypted)

Call this on every app launch and periodically while running (every few hours is fine).

```json
{
  "token": "<jwt from activation or last validation>",
  "deviceId": "<same hwid you used at activation>"
}
```

**Response:**
```json
{
  "token": "<refreshed jwt>",
  "validUntil": 1234567890
}
```

Always replace your cached token with the one from the response.

---

## Offline Grace Period

If the server is unreachable, fall back to your local cache:

```
now < validUntil              → license is fine, proceed
now < validUntil + gracePeriod → warn user, still proceed  
now >= validUntil + gracePeriod → block, tell user to connect
```

A grace period of 7 days is reasonable. Don't go longer than the token TTL (also 7 days) or the grace period becomes meaningless.

---

## Error Handling

| Status | Meaning |
|--------|---------|
| 400 | Bad request / missing encryption header |
| 401 | Invalid or expired token |
| 403 | License revoked or expired |
| 404 | License not found |
| 409 | Activation limit reached |
| 500 | Server error — fall back to cache |

On 403/404, the license is genuinely invalid. Don't retry. On 5xx or network errors, use the cached token and grace period logic.

---

## Update Checks

`POST /api/v1/updates/check` (encrypted)

Call this to check whether a newer version of your app is available. The endpoint validates your JWT before responding, so only active licensed clients can query it.

```json
{
  "token": "<jwt from activation or last validation>",
  "version": "1.2.3"
}
```

**Response:**
```json
{
  "currentVersion": "1.2.3",
  "latestVersion": "1.3.0",
  "updateAvailable": true,
  "downloadUrl": "https://github.com/..."
}
```

- `updateAvailable` is `true` when `latestVersion != currentVersion`
- `downloadUrl` points to the latest release asset if one exists, otherwise the release page

**When to call:** Once on launch, after validation succeeds. Don't call it before you have a valid token — you'll get a 401.

**On failure:** Update checks are non-critical. If this endpoint returns an error or is unreachable, log it and proceed — don't block the app.

---

## Security Notes

- **Pin the server's X25519 public key** — fetch it once during distribution, bundle it with your app. Don't re-fetch at runtime.
- **Always verify `X-Client-Key-Echo`** — if it doesn't match your ephemeral pubkey, someone is replaying an old response.
- **Encrypt your local cache** — use the OS keychain or platform-specific secure storage. Don't store the JWT in plaintext.
- **Use the same HWID consistently** — if it changes between calls, validation will fail with a 403.
