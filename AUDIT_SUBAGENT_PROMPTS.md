# Audit & Sub-Agent Prompts — account-stock-be

> Reviewed: 2026-03-18 | Reviewer: Engineering Manager (AI)
> Purpose: Project understanding + 6 self-contained prompts for Claude sub-agents to audit & fix code

---

## Part 1: Project Understanding Summary

### What is this?
**account-stock-be** — Go backend for a multi-tenant SaaS serving TikTok Shop sellers and affiliates.
Frontend counterpart: **account-stock-fe** (Next.js). Specs live in both repos.

### Tech Stack
| Layer | Tech |
|-------|------|
| Language | Go 1.21+ |
| HTTP | net/http (stdlib, no framework) |
| ORM | GORM (gorm.io/gorm) |
| DB | PostgreSQL / Supabase |
| Auth | JWT HS256 (golang-jwt/v5) |
| Password | bcrypt |
| Migration | golang-migrate (versioned SQL) |
| Deploy | Fly.io / Railway / Render |

### Architecture
```
cmd/server/main.go        → HTTP server, route registration, middleware chains
cmd/migrate/main.go       → DB migration runner
internal/auth/             → JWT (jwt.go), Roles/Tier (context.go), bcrypt (password.go)
internal/middleware/        → auth.go, permission.go, tenant.go, secure.go, cors.go
internal/handler/           → auth, users, shops, inventory, import, affiliate, analytics, dashboard, invite, self, consent
internal/model/             → GORM models: User, Shop, Company, InviteCode, TierHistory, SystemConfig
internal/rbac/              → Role→Permission mapping (rbac.go)
internal/database/          → DB connection (database.go)
migrations/                 → 000001-000015 SQL (up/down pairs, 000004 missing)
docs/                       → ENTITY_SPEC, SECURITY, DB_SPEC, DEPLOY, PRODUCTION_READINESS, feature/*.md
```

### Auth & RBAC
- **JWT claims**: Subject (user_id), Role, Tier, CompanyID, ShopID, ShopName, DisplayName
- **4 roles**: Root (superuser, no shop), SuperAdmin (shop owner), Admin (shop staff), Affiliate
- **Permissions**: `resource:action` format (e.g. `inventory:read`). Checked via `RequirePermission` middleware or inline.
- **Root bypass**: Root skips all permission checks in middleware.

### Multi-Tenant Model
- **Tenant = Shop** (shop_id). Company is parent (1 company : N shops).
- **CRITICAL RULE**: shop_id/company_id for DB scoping comes from **JWT auth context ONLY**. Never from body/query/path.
- Tables: `import_sku_row` scoped by shop_id; `affiliate_sku_row` scoped by company_id+user_id; `users` by shop_id.
- Global tables (no tenant): `invite_codes`, `system_config`, `companies`.

### Endpoints (31 routes)
| Category | Routes | Auth | Key Permissions |
|----------|--------|------|-----------------|
| Public | /health, /api/auth/login, /api/invite/validate, /api/invite/check-required, /api/consent/pdpa | None | — |
| User context | /api/auth/me, /api/users/me, /api/invite/use | JWT | — |
| Users | /api/users | JWT | users:read |
| Inventory | /api/inventory/import, /api/inventory, /api/inventory/summary | JWT | inventory:create/read |
| Shops | /api/shops, /api/shops/me, /api/shops/me/members | JWT | shops:create, users:read/create, shops:update |
| Dashboard | /api/dashboard/* (4 endpoints) | JWT | dashboard:read |
| Affiliate | /api/affiliate/import | JWT | inventory:create |
| Analytics | /api/analytics/* (5 endpoints) | JWT | analytics:read |
| Admin | /api/admin/invites, /api/admin/invites/:id, /api/admin/system-config | JWT | invites:*, config:* (inline checks) |

### DB Schema (15 migrations, no 000004)
| Table | Tenant-scoped? | Key |
|-------|---------------|-----|
| companies | No (is parent) | id PK |
| shops | No (is tenant) | id PK, company_id FK |
| users | Yes (shop_id) | id PK, shop_id FK, company_id, email unique |
| import_sku_row | Yes (shop_id) | UUID PK, unique(shop_id,date,sku_id) |
| affiliate_sku_row | Yes (company_id+user_id) | UUID PK, unique(company_id,user_id,order_id,sku_id) |
| invite_codes | No (global) | id PK, code unique |
| tier_history | Via user | id PK, user_id FK |
| system_config | No (global) | id PK, key unique |

### Known Critical Issues Found (Pre-fix)
| # | Severity | Issue | File |
|---|----------|-------|------|
| 1 | **CRITICAL** | Root login dev defaults used in production | handler/auth.go |
| 2 | **CRITICAL** | `http.Error()` bypasses safe JSON error writer | auth.go, users.go, shops.go |
| 3 | **HIGH** | Admin role missing `analytics:read` permission | rbac/rbac.go |
| 4 | **HIGH** | No "user already used this code" check (should 409) | handler/invite.go |
| 5 | **MEDIUM** | ValidateInviteCode returns 404/400 instead of 200+valid=false | handler/invite.go |
| 6 | **MEDIUM** | Down migration FK drop order wrong (tier_history before invite_codes) | 000015.down.sql |
| 7 | **LOW** | Missing 000014 down migration file | migrations/ |

---

## Part 2: Sub-Agent Prompts

> Run order: Security (1) → Tenant (2) → RBAC (3) → Migration (4) → Handler (5) → API Contract (6)
> After fixes: `go build ./...` && `go vet ./...`

---

### Sub-Agent 1: Security & Auth Hardening

```
You are a senior security engineer. Audit and FIX security bugs in this Go backend (net/http + GORM + PostgreSQL). JWT HS256 auth. 4 roles: Root/SuperAdmin/Admin/Affiliate. Error responses must use predefined constants only.

FILES: internal/auth/*.go, internal/middleware/*.go, internal/handler/auth.go, cmd/server/main.go

CRITICAL FIXES NEEDED:
1. handler/auth.go Login() — dev fallback sets default Root credentials ("superadmin"/"pass@1congrate"/"YIM2021") REGARDLESS of APP_ENV. In production with missing env vars, hardcoded defaults are used. FIX: if APP_ENV=production, refuse Root login when env vars missing.
2. handler/auth.go, users.go, shops.go — `http.Error()` bypasses safe JSON error writer (WriteJSONError). Grep and replace ALL instances.
3. /api/admin/invites and /api/admin/system-config use Auth+Tenant but inline role checks instead of RequirePermission middleware. Verify inline checks are equivalent to RBAC permissions.

ALSO CHECK:
- JWT: HS256 only (WithValidMethods), no `none`, claim length limits, MaxTokenLen=8192
- bcrypt cost factor adequate
- CORS doesn't default to "*" in production
- Login returns same 401 for wrong email and wrong password (no enumeration)
- All error responses use WriteJSONError with predefined constants only

DELIVERABLE: For each issue — state file, line, OWASP category, risk, and apply the fix.
```

---

### Sub-Agent 2: Multi-Tenant Isolation Audit

```
You are auditing tenant isolation. CRITICAL RULE: shop_id/company_id for DB scoping MUST come from auth context ONLY (middleware.GetContext(r.Context()).ShopID/.CompanyID). NEVER from body/query/path.

Tenant-scoped tables: users (shop_id), import_sku_row (shop_id), affiliate_sku_row (company_id+user_id), shops (company_id). Global: invite_codes, system_config.

FILES: ALL internal/handler/*.go, internal/middleware/tenant.go

FOR EACH HANDLER:
- [ ] Verify scope field comes from auth context, NOT request
- [ ] Every SELECT/UPDATE/DELETE on tenant table has WHERE shop_id/company_id = ?
- [ ] INSERTs set shop_id from context

KNOWN CONCERNS:
- shop_id vs company_id inconsistency: import_sku_row uses shop_id, affiliate_sku_row uses company_id. Auth middleware maps ShopID→CompanyID when empty.
- Root has CompanyID="root" (injected by middleware). Verify no DB collision.
- defaultRootShopID = "00000000-0000-0000-0000-000000000001" — verify no collision.
- shopIDsForContext() queries all shops by company_id when ShopID empty — verify correct.

DELIVERABLE: Summary table (handler → table → scope field → correct?) + fix violations.
```

---

### Sub-Agent 3: RBAC Permission Matrix Audit

```
You are verifying RBAC enforcement. 4 roles: Root (bypasses all), SuperAdmin, Admin, Affiliate. Permissions = resource:action.

FILES: internal/rbac/rbac.go, cmd/server/main.go, handlers with inline checks

KNOWN ISSUE: Admin role MISSING analytics:read in rbac.go (has analysis:read but NOT analytics:read). Routes /api/analytics/* require analytics:read → Admin gets 403. FIX: add PermAnalyticsRead to Admin's permissions.

ALSO VERIFY:
- Admin endpoints (/api/admin/invites, /api/admin/system-config) use inline role checks NOT RequirePermission. Verify equivalence. Config should be Root-only.
- /api/shops/me inline checks: GET→users:read, PATCH→shops:update. No RequirePermission in middleware chain — verify no bypass.
- SuperAdmin should NOT have shops:create, shops:delete, config:read, config:update.

DELIVERABLE: Permission matrix diff (expected vs actual per route). Fix mismatches.
```

---

### Sub-Agent 4: Migration Integrity & Schema Consistency

```
You are auditing SQL migrations (000001-000015, note: 000004 missing) and GORM models.

MIGRATIONS: migrations/*.sql (read all up AND down files)
MODELS: internal/model/*.go + inline structs in internal/handler/inventory.go, affiliate.go, analytics.go, dashboard.go, import.go

CRITICAL FIXES:
1. 000015.down.sql — drops tier_history before invite_codes, but tier_history.invite_code_id has FK to invite_codes.id. Reorder: drop tier_history first (it IS first currently — verify the FK actually exists and if system_config needs ordering too).
2. 000014 may be missing .down.sql file — verify and create if needed.

ALSO CHECK:
- Each .down.sql correctly reverses .up.sql
- Final schema matches GORM models (user.go has phone? invite_code_used VARCHAR(32) vs VARCHAR(36)?)
- affiliate_sku_row has both `commission` (000008) and `commission_amount` (000009) — is `commission` still used?
- Unique indexes match upsert logic in handlers
- SQL uses NUMERIC(18,2) but Go uses float64 — note precision risk

DELIVERABLE: Schema diff table + fix model/migration mismatches.
```

---

### Sub-Agent 5: Handler Logic & Business Rule Correctness

```
You are reviewing ALL handler implementations for bugs and spec compliance.

FILES: ALL internal/handler/*.go (11 files)
SPECS: docs/feature/01-auth.md through 06-invites.md

KNOWN BUGS:
1. invite.go UseInviteCode — missing "user already used this code" check. Per spec (06-invites.md), should return 409 if user.invite_code_used == req.Code. Current code only checks invite validity.
2. invite.go ValidateInviteCode — returns 404/400 for invalid codes instead of 200 with {valid:false} per spec.

ALSO CHECK:
- Every handler checks r.Method
- Every handler uses WriteJSONError (not http.Error)
- database.DB() == nil handled gracefully everywhere
- Root user with empty ShopID accessing tenant-scoped endpoints — what happens?
- Timezone: analytics uses Asia/Bangkok, other code uses UTC — any inconsistency?
- shops.go ShopsMeMembers: verify no SuperAdmin creation allowed via POST
- analytics.go: verify all 5 endpoints handle both seller and affiliate paths

DELIVERABLE: Bug list with severity + fix each directly.
```

---

### Sub-Agent 6: API Contract & Frontend Alignment

```
You are verifying backend response shapes match frontend expectations.

BACKEND: internal/handler/*.go
FRONTEND: account-fe/contexts/AuthContext.tsx, account-fe/lib/

KEY CONTRACTS:
- GET /api/auth/me → {user:{id,displayName}, roles:[], permissions:[], tier, tier_started_at, tier_expires_at, invite_code_used, invite_slots, company_id, shop_id(null for Root), shop_name}
- POST /api/auth/login → {token: "JWT"}
- GET /api/admin/invites → response key "codes" vs spec "invites" — MISMATCH. Fix to match frontend.
- POST /api/admin/invites → wraps in {code: {...}} — verify frontend expects this
- POST /api/invite/validate → spec says return {valid, code, grant_tier, tier_duration_days, remaining_uses, expires_at} but handler returns subset

CHECK:
- JSON field naming: snake_case vs camelCase consistency
- Status codes match spec (200, 201, 400, 401, 403, 404, 405, 409, 500)
- Error shape always {error: "message"} (never raw text)
- Content-Type: application/json on all responses

DELIVERABLE: Contract diff table (endpoint → expected → actual → match?) + fix mismatches.
```

---

## How to Use

1. Run each sub-agent independently — they are self-contained
2. Order: Security (1) → Tenant (2) → RBAC (3) → Migration (4) → Handler (5) → API (6)
3. After fixes: `go build ./...` && `go vet ./...`
4. Final pass: re-run all agents on fixed code to verify no regressions

