# account-stock-be

Backend for **account-stock-fe** (Next.js). Go (Golang) API with auth, RBAC, and multi-tenant (company_id) per `project-specific_context.md` and frontend specs.

## Requirements

- Go 1.22+

## Run

```bash
go mod tidy   # if needed
go run ./cmd/server
```

Server listens on `:8080`.

## Database (PostgreSQL or Supabase)

- **Same driver:** GORM + `gorm.io/driver/postgres` — works for both self-hosted Postgres and Supabase (managed Postgres).
- **Config:** Set `DATABASE_URL` or `SUPABASE_DB_URL`. When set, server connects at startup; when unset, server runs without DB. See `.env.example`.
- **ต่อ Supabase:** ขั้นตอนครบใน **`docs/SUPABASE.md`** (สร้างโปรเจกต์, เอา connection string, รัน migrate, รัน server).
- **Schema migration:** From repo root: `go run ./cmd/migrate`. Uses `migrations/*.up.sql`; reads `DATABASE_URL` or `SUPABASE_DB_URL`. See **`docs/DB_SPEC.md`**.

## Endpoints

| Method | Path | Auth | Permission | Description |
|--------|------|------|------------|-------------|
| GET | `/health` | no | — | Health check |
| GET | `/api/auth/me` | JWT Bearer | — | Current user context (user, roles, permissions, tier, company_id) |
| GET | `/api/users` | JWT Bearer | `users:read` | List users (SuperAdmin only) |

## Auth (JWT)

- **Authorization:** `Bearer <JWT>`. JWT must contain: `sub` (user id), `role`, `tier`, `company_id`, optional `display_name`.
- **Config (env):** `JWT_SECRET` (required in prod), optional `JWT_ISSUER`, `JWT_AUDIENCE`. Default secret is **dev-only** — do not use in production.
- **Permissions:** Derived from `role` on backend (RBAC_SPEC); no need to send permissions in token.

## Security (OWASP Top 10 & injection)

- **Error responses:** Fixed messages only; JSON via `encoding/json` (no injection from user input).
- **JWT claims:** Role/tier allowlist; claim length limits; token length limit (8KB).
- **Access control:** RBAC per route; unknown role → 401.
- Full checklist: **`docs/SECURITY.md`**.

## Structure

- `cmd/server` — main entry, routes, middleware chain (Auth → RequirePermission where needed)
- `internal/auth` — Context, Role, Tier, JWT claims and ValidateToken
- `internal/rbac` — role→permissions map, HasPermission (RBAC_SPEC §5)
- `internal/middleware` — Auth(JWT), RequirePermission(permission), RequireAuthContext, Tenant, secure error responses
- `internal/handler` — Me (/api/auth/me), UsersList (/api/users)
- `internal/database` — GORM connection (Postgres/Supabase), config from env
- `internal/model` — GORM models (Company, User with company_id)
- `migrations/` — Versioned SQL (000001_init.up/down.sql); run via `go run ./cmd/migrate`
- `cmd/migrate` — Runs migrations (DATABASE_URL or SUPABASE_DB_URL)

## Align with frontend

- API contract and user context: see `project-specific_context.md`
- Specs: `account-stock-fe/docs/USER_SPEC.md`, `account-stock-fe/docs/RBAC_SPEC.md`

## TODO

- Wire DB into handlers (users list from DB with TenantScope); see `internal/database/scope.go`
- Audit log (userId, resource, action, result, timestamp)
- Login endpoint that issues JWT (frontend currently uses dev login; backend can issue token after credential check)
