# Security — OWASP Top 10 & Injection

Backend measures to align with **OWASP Top 10 (2021)** and prevent **injection** from user-supplied values.

---

## A01:2021 – Broken Access Control

- **RBAC:** Every protected route is gated by `Auth` (JWT) and, where needed, `RequirePermission(permission)`. Permissions are derived from **role allowlist** only (`ValidRole`); unknown role → 401.
- **Deny by default:** Unknown role in JWT is rejected (invalid token). No fallback to elevated permissions.
- **Multi-tenant:** Handlers must use `company_id` from auth context for all tenant-scoped data (no client-supplied company_id in body).

---

## A02:2021 – Cryptographic Failures

- **JWT:** HS256 with server-held secret (`JWT_SECRET`). Algorithm restricted via `WithValidMethods` (no `none`/alg confusion).
- **Secrets:** Load from env; no hardcoded production secrets. Default dev secret must not be used in production (see README).
- **TLS:** Use HTTPS in production; this service does not terminate TLS (use reverse proxy/load balancer).

---

## A03:2021 – Injection

- **Error responses:** All API error bodies use fixed messages and `encoding/json` (no string concatenation of user input). See `middleware/secure.go`: `WriteJSONError` with constants only.
- **JWT claims:** Role/tier validated with **allowlists** (`ValidRole`, `ValidTier`). Claim lengths capped (`ValidateClaimLengths`, `MaxTokenLen`) to limit DoS and injection surface.
- **Future:** Any SQL/NoSQL must use parameterized queries; no concatenation of user input into queries or commands. Logging must not pass unsanitized user input into log format strings.

---

## A04:2021 – Insecure Design

- **Claim limits:** `MaxClaimSubjectLen`, `MaxClaimCompanyIDLen`, `MaxClaimDisplayNameLen` (256), `MaxTokenLen` (8KB) to prevent oversized payloads.
- **Auth flow:** JWT required for protected routes; no optional auth that could bypass checks.

---

## A05:2021 – Security Misconfiguration

- **Default JWT secret:** Only for local dev; production must set `JWT_SECRET` (and optionally `JWT_ISSUER`, `JWT_AUDIENCE`).
- **Headers:** `Content-Type: application/json` where applicable; no sensitive data in error details.

---

## A06:2021 – Vulnerable and Outdated Components

- Keep `go.mod` dependencies up to date; run `go list -m -u all` and security advisories (e.g. `govulncheck`) periodically.

---

## A07:2021 – Identification and Authentication Failures

- **JWT:** Required for `/api/auth/me` and `/api/users`; invalid/expired token → 401. No detailed error to client (generic “invalid or expired token”).
- **Role/tier:** Only allowlisted values accepted; otherwise 401.

---

## A08:2021 – Software and Data Integrity Failures

- JWT signature verified with server secret; algorithm fixed to HS256. Dependencies from trusted module proxy.

---

## A09:2021 – Security Logging and Monitoring Failures

- **TODO:** Audit log for access (userId, resource, action, result, timestamp) per RBAC_SPEC. Do not log tokens or passwords.

---

## A10:2021 – Server-Side Request Forgery (SSRF)

- No outbound requests to user-supplied URLs in current code. If added, validate/allowlist targets and avoid forwarding client-controlled URLs.

---

## Injection Prevention Checklist

| Source of value        | Mitigation |
|------------------------|------------|
| Authorization header   | ParseBearer; max length; JWT verify; claims allowlist + length check |
| Error message to client| Predefined constants only; body via `json.Encoder` |
| Role / tier in JWT     | `ValidRole`, `ValidTier` allowlist |
| Response body (e.g. /me)| `json.Encoder.Encode` (escapes strings) |
| Future: query/body params | Parameterized queries; no concat into SQL/shell |
