# Entity Spec — User, Tenant (Company) และการ inject tenant

Spec สำหรับ **entity ฝั่ง backend** และกฎ **tenant injection** (scope ตาม company_id). อ่านก่อน implement ฟีเจอร์ที่เกี่ยวกับ user/company หรือข้อมูลแยกตามเจ้า. สอดคล้องกับ `account-stock-fe/docs/USER_SPEC.md` และ `project-specific_context.md`.

---

## 1. Scope

- กำหนด **entity หลัก** ที่เกี่ยวกับ user และ tenant (company) ใน DB.
- กำหนด **ที่มาและกฎของ tenant scope** — ข้อมูลแยกตามเจ้าต้องใช้ `company_id` จากที่ใดเท่านั้น (inject จาก auth context; ห้ามใช้ค่าจาก client เป็น scope).

---

## 2. Entities

### 2.1 Company (tenant / เจ้า)

| ฟิลด์ | Type | ข้อกำหนด | หมายเหตุ |
|--------|------|-----------|----------|
| `id` | VARCHAR(36) | PK, NOT NULL | UUID หรือ unique string |
| `name` | VARCHAR(256) | NOT NULL | ชื่อบริษัท/เจ้า |
| `created_at` | TIMESTAMPTZ | NOT NULL | |
| `updated_at` | TIMESTAMPTZ | NOT NULL | |
| `deleted_at` | TIMESTAMPTZ | nullable | soft delete |

- **ไม่ใช่ tenant-scoped** (ไม่มี `company_id` ในตารางนี้) — เป็นตัวกำหนด tenant เอง.

### 2.2 User

| ฟิลด์ | Type | ข้อกำหนด | หมายเหตุ |
|--------|------|-----------|----------|
| `id` | VARCHAR(36) | PK, NOT NULL | ตรงกับ user_id ใน auth context |
| `company_id` | VARCHAR(36) | NOT NULL, index | **Tenant scope** — ต้องตรงกับ company ที่ user สังกัด |
| `email` | VARCHAR(256) | unique (where deleted_at IS NULL) | |
| `display_name` | VARCHAR(256) | nullable | |
| `role` | VARCHAR(32) | NOT NULL | RBAC: SuperAdmin, Admin, Manager, Staff, Viewer |
| `tier` | VARCHAR(16) | NOT NULL | free / paid (USER_SPEC) |
| `created_at` | TIMESTAMPTZ | NOT NULL | |
| `updated_at` | TIMESTAMPTZ | NOT NULL | |
| `deleted_at` | TIMESTAMPTZ | nullable | soft delete |

- **เป็น tenant-scoped:** ทุก query/update ที่เกี่ยวกับ user (ยกเว้น SuperAdmin จัดการข้ามเจ้า ตาม RBAC) ต้อง filter ตาม `company_id` ที่ได้จาก **auth context** เท่านั้น.

---

## 3. ER (ความสัมพันธ์)

```
┌─────────────────┐         ┌─────────────────┐
│    companies    │         │      users      │
├─────────────────┤         ├─────────────────┤
│ id (PK)         │◄────────│ company_id (FK) │
│ name            │    1  * │ id (PK)         │
│ created_at      │         │ email           │
│ updated_at      │         │ display_name    │
│ deleted_at      │         │ role, tier      │
└─────────────────┘         │ created_at      │
                             │ updated_at      │
                             │ deleted_at      │
                             └─────────────────┘
```

- **Company 1 — * User:** User หนึ่งคนสังกัด company เดียว (`users.company_id` → `companies.id`).
- ตารางอื่นที่แยกตามเจ้าในอนาคต (เช่น inventory, orders) ต้องมี `company_id` และความสัมพันธ์กับ `companies` ในทำนองเดียวกัน.

---

## 4. Tenant injection (กฎการใส่ / ใช้ company_id)

- **ที่มา company_id ฝั่ง backend:** มาจาก **auth context เท่านั้น** (ที่ middleware ดึงจาก JWT/session หลัง login). ไม่ใช้ค่าจาก request body หรือ query parameter เป็น **scope** ของข้อมูล.
- **การ inject:**  
  - ใน handler ที่เข้าถึงข้อมูล tenant-scoped ให้ใช้ `middleware.GetContext(r.Context()).CompanyID` (หรือ helper เช่น `database.TenantScope(companyID)`) เป็นค่าเดียวที่ใช้ filter / set ใน query หรือ insert/update.
- **สิ่งที่ห้าม:**  
  - ห้ามใช้ `company_id` จาก JSON body หรือ query string เพื่อกำหนดว่า “ดู/แก้ข้อมูลของเจ้าไหน”.  
  - ห้ามข้าม middleware auth แล้วไปอ่าน company_id จากที่อื่น.
- **ข้อยกเว้น (SuperAdmin):** ฟีเจอร์ที่ RBAC กำหนดให้ SuperAdmin จัดการข้าม tenant (เช่น list users ทุก company) ต้องมี permission เฉพาะและออกแบบ endpoint แยก (เช่น query param สำหรับ filter company ได้เฉพาะเมื่อ role = SuperAdmin และตรวจที่ backend).

---

## 5. ตารางที่ถือว่า tenant-scoped (ปัจจุบันและอนาคต)

| ตาราง | มี company_id | หมายเหตุ |
|--------|----------------|----------|
| `companies` | ไม่มี | เป็นตัว tenant เอง |
| `users` | มี | scope ตาม company ที่ user สังกัด |
| (อนาคต) inventory, orders, … | มี | ต้องใช้ company_id จาก auth context เท่านั้น |

---

## 6. Acceptance criteria

| ID | Criteria | วิธีตรวจ |
|----|----------|----------|
| AC1 | ทุก entity ตาม spec มีฟิลด์และ constraint ตรงกับตารางด้านบน | เปรียบเทียบ migration / model กับ spec |
| AC2 | ทุก query/update ที่เป็น tenant-scoped ใช้ company_id จาก auth context เท่านั้น | Code review; ไม่มีการอ่าน company_id จาก body/query สำหรับ scope |
| AC3 | มี ER หรือเอกสารความสัมพันธ์ Company — User | ตรวจ docs/ENTITY_SPEC.md และ migration |

---

## 7. อ้างอิง

- **User context (ความหมาย role, tier, company):** `account-stock-fe/docs/USER_SPEC.md`
- **RBAC (role, permission):** `account-stock-fe/docs/RBAC_SPEC.md`
- **Backend context & API:** this repo `project-specific_context.md`
- **DB & migration:** this repo `docs/DB_SPEC.md`
