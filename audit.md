Security Review — clave Backend

  Vuln 1: Unauthenticated License Creation — internal/api/router.go:21

  - Severity: High
  - Confidence: 9/10
  - Category: auth_bypass
  - Description: POST /api/v1/ (CreateLicense) has no authentication middleware. The only middleware on the v1Router group is chi's built-in RequestID, Recoverer, Timeout, and Logger. RequireSelfServiceAuth is applied only to the nested /selfservice sub-router. Any
   unauthenticated caller can create licenses.
  - Exploit Scenario: Attacker sends POST /api/v1/ with {"productId":1,"maxActivations":9999}. Server generates a valid, database-backed license key with unlimited activations — no credentials required. Entire monetisation model is bypassed.
  - Recommendation: Gate CreateLicense (and all admin operations) behind an API key or admin JWT middleware. Apply it at the route registration level in router.go.

  ---
  Vuln 2: Revoked Licenses Not Checked — internal/services/activation.go, validation.go

  - Severity: High
  - Confidence: 9/10
  - Category: auth_bypass
  - Description: domain.License.IsActive is fetched from the DB and mapped correctly, but neither ActivationService.Activate nor ValidationService.Validate ever checks it. A license with is_active = false passes all checks — key verification, activation count,
  expiry — and continues to receive tokens.
  - Exploit Scenario: Admin revokes a license in the DB (is_active = false). The customer's client continues activating and calling /validate indefinitely, receiving fresh 7-day tokens. Revocation has zero effect.
  - Recommendation: Add an early guard in both Activate and Validate:
  if !license.IsActive {
      return ..., problem.Of(403).Append(problem.Title("License revoked"))
  }

  ---
  Vuln 3: HWID Device-Binding Check Bypassed by Omitting deviceId — internal/services/validation.go:48

  - Severity: High
  - Confidence: 9/10
  - Category: auth_bypass
  - Description: The HWID check is if data.DeviceID != "" && claims.HWID != "" && data.DeviceID != claims.HWID. The DeviceID field in LicenseValidationRequest has no validate:"required" tag. Omitting deviceId from the JSON body leaves data.DeviceID == "",
  short-circuiting the entire check. Any client can validate a device-locked license token without presenting a matching HWID.
  - Exploit Scenario: Attacker steals a JWT issued to Device A (e.g., from a memory dump, log, or network capture). They POST {"token":"<stolen>"} to /validate with no deviceId. The HWID check is skipped; a fresh 7-day token is issued bound to no device. Attacker
  now has indefinitely renewable access from any machine.
  - Recommendation: Either (a) require DeviceID in the validation DTO (validate:"required"), or (b) change the check to if claims.HWID != "" && data.DeviceID != claims.HWID — i.e., reject validation when the token has an HWID but the request doesn't provide one.

  ---
  Vuln 4: Token Renewal Does Not Check Revocation or Activation Record — internal/services/validation.go:55–68

  - Severity: High
  - Confidence: 9/10
  - Category: auth_bypass
  - Description: Distinct from Vuln 2 (activation path). Validate independently has no IsActive check and no check that the originating activation record still exists. A customer who was revoked after activation — and holds a valid JWT — calls /validate every 7
  days to renew. The server finds the license row, sees it exists, skips the active check, and issues a fresh token forever.
  - Exploit Scenario: Chained with Vuln 2: revoke a license at DB level; the customer's token renews automatically on each 7-day cycle, giving permanent access post-revocation with no way to stop it short of waiting for all tokens to naturally expire (up to 7 days
  per cycle, indefinitely renewable).
  - Recommendation: Add IsActive check in ValidationService.Validate (see Vuln 2 recommendation). Optionally also verify the activation record still exists in the activations table.

  ---
  Vuln 5: Self-Service Magic-Link Token Returned in HTTP Response Body — internal/handlers/selfservice.go:39

  - Severity: Medium
  - Confidence: 9/10
  - Category: data_exposure
  - Description: The raw self-service token is returned directly in the HTTP JSON response (SelfServiceRequestLinkResponse{Ok: true, Token: rawToken}). There is no email-sending logic anywhere in the codebase — the token is never sent out-of-band. This makes the
  "magic link" flow fully in-band, exposing the token to any observer of the HTTP response (proxies, CDN logs, browser extensions, network sniffers).
  - Exploit Scenario: Attacker with access to HTTP access logs (common in cloud deployments), a shared network, or any proxy between client and server reads the response body, extracts the token, and immediately POSTs it to the validate endpoint — gaining a full
  self-service JWT for the victim's email address and access to all their license data.
  - Recommendation: Remove Token from the HTTP response. Implement out-of-band delivery (email via SMTP/SES). The token should only ever exist in the DB (hashed) and in the email inbox of the owner.
