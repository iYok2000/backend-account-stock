# Agent prompt — account-stock-be

You are working on **account-stock-be** — the backend for **account-stock-fe** (Next.js frontend). Follow these instructions.

## Spec-first

- **Start from spec.** Before implementing, read the relevant spec: `account-stock-fe/docs/USER_SPEC.md`, `account-stock-fe/docs/RBAC_SPEC.md`, this repo’s `project-specific_context.md`, and when touching **user/tenant or DB entities**: this repo’s **`docs/ENTITY_SPEC.md`** (User, Company, tenant injection, ER). Design and API must follow the spec.
- **Gaps:** If the spec is missing or unclear, clarify or document (e.g. in a doc or ticket) before coding. Do not assume behaviour that contradicts the spec.

## Before any task

1. **Read project context:** `project-specific_context.md` — API contract, auth, RBAC, multi-tenant, and structure are mandatory.
2. **Read relevant spec:** USER_SPEC, RBAC_SPEC (frontend repo); **ENTITY_SPEC** (this repo) for user/company entity and tenant injection; **`docs/feature/*.md`** for the API/feature you are changing (auth, users, …); DB_SPEC for migrations.
3. **Check skills:** If the task matches a skill in `.cursor/skills/`, read that skill’s `SKILL.md` and follow it.

## Acceptance criteria

- **Define or reuse.** For each task, identify acceptance criteria (from spec, ticket, or product requirement). If none exist, state them before implementation.
- **Verify before done.** Before considering the task complete, confirm that the implementation meets the acceptance criteria (e.g. endpoints, status codes, permission checks, tenant scope).
- **New behaviour:** When adding features, document acceptance criteria (in spec or PR/ticket) so they can be checked and reviewed.

## Security review

- **Part of the workflow.** For every change that touches auth, input, or output: consider security (OWASP Top 10, injection, access control).
- **Backend checklist:** Use this repo’s `docs/SECURITY.md`: error responses (fixed messages, JSON-encoded), JWT claims (allowlist, length limits), RBAC enforcement, no user input in error bodies, parameterized queries when using DB.
- **Do not** rely on frontend for enforcement; backend must validate and enforce permissions and tenant scope.

## Rules

- **Align with frontend.** API shape, auth (user context: `user_id`, `role`, `tier`, `company_id`), RBAC (resource:action), and multi-tenant (`company_id` scope) must match `account-stock-fe` and the specs in the frontend repo (`docs/USER_SPEC.md`, `docs/RBAC_SPEC.md`).
- **No cross-domain violations.** Handlers and domain logic must not bypass middleware (auth, permission, tenant scope). Shared code lives in middleware/lib; domain-specific in respective packages.
- **Enforce at backend.** Permission checks and `company_id` scoping are mandatory for every relevant request. Do not rely on frontend for security.
- Before starting: search memory for relevant conventions, patterns, and decisions.
- After significant decisions: store memory (title, content, tags, scope).

## References (read when relevant)

- **User & multi-tenant (ความหมาย context):** Frontend `docs/USER_SPEC.md` — user context fields, tier, company_id, backend responsibilities.
- **User & Tenant (entity + inject):** This repo **`docs/ENTITY_SPEC.md`** — User/Company entity, ER, กฎ tenant injection (company_id จาก auth context เท่านั้น; ห้ามใช้จาก client เป็น scope).
- **RBAC:** Frontend `docs/RBAC_SPEC.md` — roles, resources, actions, permission matrix, backend enforcement, audit log, testing.
- **Backend context:** This repo `project-specific_context.md` — structure, middleware, API contract, auth flow.
- **Security:** This repo `docs/SECURITY.md` — OWASP Top 10, injection prevention, error handling.
- **Database:** This repo `docs/DB_SPEC.md` — GORM, Postgres/Supabase, migration, multi-tenant scope.
- **Feature API:** This repo `docs/feature/README.md` + `docs/feature/*.md` — route, permission, request/response, tenant scope, acceptance criteria ต่อฟีเจอร์ (auth, users, …).
