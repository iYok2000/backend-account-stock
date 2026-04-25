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

- **Error responses:** All API error bodies use fixed constants (`ErrInternal`, `ErrForbidden`, etc.) from `middleware/secure.go`. Raw `err.Error()` is never sent to the client.
- **JWT claims:** Role/tier validated with **allowlists** (`ValidRole`, `ValidTier`). Claim lengths capped (`ValidateClaimLengths`, `MaxTokenLen`) to limit DoS and injection surface.
- **SQL:** All DB queries use GORM parameterized queries; no user input is concatenated into SQL strings.

---

## A04:2021 – Insecure Design

- **Request body limit:** Global `limitBodySize` middleware caps all request bodies at **32 MB** to prevent memory DoS from oversized payloads (`cmd/server/main.go`).
- **HTTP timeouts:** `ReadHeaderTimeout=10s`, `ReadTimeout=30s`, `WriteTimeout=60s`, `IdleTimeout=120s` to prevent Slowloris-style connection exhaustion.
- **Claim limits:** `MaxClaimSubjectLen`, `MaxClaimCompanyIDLen`, `MaxClaimDisplayNameLen` (256), `MaxTokenLen` (8KB) to prevent oversized payloads.
- **Auth flow:** JWT required for protected routes; no optional auth that could bypass checks.

---

## A05:2021 – Security Misconfiguration

- **Default JWT secret:** Only for local dev; production must set `JWT_SECRET` (and optionally `JWT_ISSUER`, `JWT_AUDIENCE`).
- **CORS:** `Access-Control-Allow-Credentials: true` is only set when a specific origin matches — never sent alongside `Access-Control-Allow-Origin: *`.
- **Headers:** `Content-Type: application/json` where applicable; no sensitive data in error details.

---

## A06:2021 – Vulnerable and Outdated Components

- Keep `go.mod` dependencies up to date; run `go list -m -u all` and security advisories (e.g. `govulncheck`) periodically.

---

## A07:2021 – Identification and Authentication Failures

- **JWT:** Required for `/api/auth/me` and all protected endpoints; invalid/expired token → 401. No detailed error to client (generic "invalid or expired token").
- **Root credential comparison:** Uses `crypto/subtle.ConstantTimeCompare` for email, password, and confirm code to prevent timing-based credential enumeration.
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

---

## Pitfalls — JWT/Auth (ต้องเข้มงวดในการ implement)

**สรุปจากวิจัย (JWT authorization, OWASP):**

- **Algorithm confusion:** ห้ามใช้ค่า `alg` จาก header ของ JWT ในการ verify โดยไม่จำกัด. กำหนด algorithm ที่ยอมรับแบบ hardcode/whitelist เท่านั้น; ไม่รับ `none`; ไม่ผสม HMAC กับ asymmetric ใน path เดียว (A02).
- **ตรวจ claim ครบ:** ตรวจ signature + `exp` (และ `iss`/`aud` ถ้าใช้); อนุญาต clock skew เล็กน้อย. เช็คแค่ signature = token หมดอายุหรือ issuer ผิดยังใช้ได้.
- **Secret ใน production:** Production บังคับ JWT_SECRET ที่ไม่ใช่ default; อ่านจาก env เท่านั้น (DEPLOY.md).

**กรณีที่อนาคตอาจกระทบ:** Endpoint ใหม่ต้องผ่าน Auth; ถ้าเป็น resource ต้องมี RequirePermission. ดู ENTITY_SPEC §7 สำหรับ tenant scope และ endpoint/table/job/cache.
