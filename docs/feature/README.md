# Feature Specs — Backend API ต่อฟีเจอร์

เอกสารในโฟลเดอร์นี้อธิบาย **API ต่อ feature** (route, permission, request/response, tenant scope) เพื่อให้ implement ตาม spec-first และสอดคล้องกับ frontend `docs/feature/*.md`.

---

## ทำไมต้องมี

- **Spec-first (AGENTS.md):** ก่อนเขียน handler/endpoint ต้องมี spec กำหนด route, permission, และ scope.
- **Acceptance criteria:** แต่ละ feature มีเกณฑ์ให้ตรวจก่อนถือว่าเสร็จ.
- **สอดคล้อง frontend:** หน้าใน fe (เช่น `/users`, `/inventory`) เรียก API ตามที่ระบุใน spec ฝั่ง be.

---

## โครงไฟล์

| ไฟล์ | Feature | API ที่เกี่ยวข้อง |
|------|---------|-------------------|
| [01-auth.md](./01-auth.md) | Auth / session | `GET /api/auth/me` |
| [02-users.md](./02-users.md) | จัดการผู้ใช้ (SuperAdmin) | `GET /api/users` (และ CRUD เมื่อมี) |
| [03-import.md](./03-import.md) | Import → Inventory | `POST /api/inventory/import`, `GET /api/inventory`, `GET /api/inventory/summary` (Auth + inventory:create/read) |
| [05-dashboard.md](./05-dashboard.md) | Dashboard | `GET /api/dashboard/overview`, `/kpis`, `/revenue-7d`, `/low-stock` (Auth + dashboard:read) |
| [06-analytics.md](./06-analytics.md) | Analytics | `GET /api/analytics/reconciliation`, `/daily-metrics`, `/product-metrics`, `/trends`, `/profitability` (Auth + analytics:read) |
| [06-invites.md](./06-invites.md) | Invite System (Tier) | `POST /api/invite/validate`, `/api/admin/invites`, `/api/admin/system-config` (Public, Auth, Admin) |
| [07-calculator.md](./07-calculator.md) | Fee Calculator | `POST /api/calculator/fees`, `/api/calculator/fees/batch` (Auth + analytics:read) |

ฟีเจอร์อื่น (inventory, orders, suppliers, …) จะเพิ่มเมื่อมี endpoint นั้นใน backend.

---

## โครงร่าง spec ต่อ feature (ให้ครบ)

- **Route & Method:** path + HTTP method
- **Permission:** resource:action (หรือ public)
- **Request:** header (auth), body (ถ้ามี)
- **Response:** shape, status code
- **Tenant scope:** ใช้ company_id จาก auth หรือไม่ (ตาม ENTITY_SPEC)
- **Acceptance criteria:** อย่างน้อย 1–3 ข้อ
