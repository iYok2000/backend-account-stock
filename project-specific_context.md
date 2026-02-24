# Project-specific context (backend — สำหรับ AI)

อ่านไฟล์นี้ก่อนทำ task — เป็นกฎและบริบทของ backend ให้สอดคล้องกับ **account-stock-fe**

---

## ความสัมพันธ์กับ Frontend

- **account-stock-be** เป็นหลังบ้านของ **account-stock-fe** (Next.js, i18n, Tailwind).
- Spec หลักอยู่ที่ frontend repo: `docs/USER_SPEC.md`, `docs/RBAC_SPEC.md` — backend ต้อง implement ให้ตรงกับ spec เหล่านั้น.

---

## API สัญญา (API contract)

- Frontend ส่ง request พร้อม **auth header** (Bearer token หรือ cookie/session).
- **ไม่รับ company ใน body** — backend ดึง `company_id` จาก user context (token/session) เสมอ.
- **`/api/auth/me` (หรือเทียบเท่า):** response ต้องมี `user.id`, `user.displayName`, `roles`, `permissions`, `tier`, `company_id` เพื่อให้ frontend ใช้กับ AuthContext / useUserContext.

---

## User context (จาก token/session)

ทุก request ที่ผ่าน auth ต้องได้ค่าดังนี้จาก middleware:

| ฟิลด์ | การใช้ |
|--------|--------|
| **user_id** | ระบุผู้ใช้, audit log |
| **role** | ตรวจสิทธิ์ตาม RBAC (SuperAdmin, Admin, Manager, Staff, Viewer) |
| **tier** | จำกัดฟีเจอร์ตามระดับบริการ (free/paid) — ตรวจที่ backend เท่านั้น |
| **company_id** | scope ข้อมูล multi-tenant — ทุก query/upsert ที่แยกตามเจ้าต้องใช้ค่านี้ |

---

## RBAC และ Multi-tenant

- **Permission:** รูปแบบ `resource:action` (เช่น `inventory:read`, `users:read`). ต้องตรวจทุก endpoint ว่าผู้ใช้มี permission นั้น — Deny by Default.
- **Multi-tenant:** ตารางที่แยกตามเจ้าต้องมี `company_id`; ทุก SELECT/UPDATE/INSERT/DELETE ต้อง scope ตาม `company_id` ของ user ที่ล็อกอิน (ไม่ให้เจ้า A เห็น/แก้ข้อมูลเจ้า B).
- **Tier:** ตรวจที่ backend เท่านั้น — ไม่พึ่ง frontend.

---

## โครงสร้างและกฎการ implement

- **Middleware / Interceptor (ชั้นกลาง):**  
  (1) ดึง user context จาก token/session  
  (2) ตรวจ permission ตาม resource:action สำหรับ route นั้น  
  (3) inject / ตรวจ `company_id` ให้ทุกการเข้าถึงข้อมูลที่แยกตาม tenant  
  ห้าม handler ข้ามหรือปิด middleware เหล่านี้.
- **Domain:** แยก handler/package ตาม domain (auth, users, inventory, orders, …). Logic ร่วมอยู่ที่ middleware และ lib; domain ไม่ข้าม domain โดยตรงถ้าผูกกับสิทธิ์/tenant โดยไม่ผ่านชั้นกลาง.
- **Audit log:** บันทึก `userId`, `resource`, `action`, `result` (allowed/denied), `timestamp` ตาม RBAC_SPEC.
- **Testing:** Unit/Integration tests สำหรับ enforcement สิทธิ์และ scope `company_id` (เช่น request ไม่มี permission ต้องถูก deny, request ข้าม company ต้องถูก deny).

---

## อ้างอิง Spec

- **User & multi-tenant (ความหมาย context):** `account-stock-fe/docs/USER_SPEC.md`
- **User & Tenant (entity, ER, กฎ inject company_id):** this repo `docs/ENTITY_SPEC.md`
- **RBAC, roles, resources, matrix, security, testing:** `account-stock-fe/docs/RBAC_SPEC.md`
- **Feature API (route, permission, response, tenant):** this repo `docs/feature/README.md`, `docs/feature/*.md`
