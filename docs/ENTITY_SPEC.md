# Entity Spec — Shop, User และการ inject tenant (shop_id)

Spec สำหรับ **entity ฝั่ง backend** และกฎ **tenant injection** (scope ตาม shop_id). อ่านก่อน implement ฟีเจอร์ที่เกี่ยวกับ user/shop หรือข้อมูลแยกตามร้าน. สอดคล้องกับ `account-stock-fe/docs/USER_SPEC.md`, `docs/SHOPS_AND_ROLES_SPEC.md` และ `project-specific_context.md`.

---

## 1. Scope

- กำหนด **entity หลัก** ที่เกี่ยวกับ user และ tenant (shop) ใน DB.
- กำหนด **ที่มาและกฎของ tenant scope** — ข้อมูลแยกตามร้านต้องใช้ `shop_id` จากที่ใดเท่านั้น (inject จาก auth context; ห้ามใช้ค่าจาก client เป็น scope).

---

## 2. Entities

### 2.1 Shop (tenant / ร้าน)

| ฟิลด์ | Type | ข้อกำหนด | หมายเหตุ |
|--------|------|-----------|----------|
| `id` | VARCHAR(36) | PK, NOT NULL | UUID หรือ unique string |
| `name` | VARCHAR(256) | NOT NULL | ชื่อร้าน |
| `created_at` | TIMESTAMPTZ | NOT NULL | |
| `updated_at` | TIMESTAMPTZ | NOT NULL | |
| `deleted_at` | TIMESTAMPTZ | nullable | soft delete |

- **ไม่ใช่ tenant-scoped** (ไม่มี shop_id ในตารางนี้) — เป็นตัวกำหนด tenant เอง.

### 2.2 User

| ฟิลด์ | Type | ข้อกำหนด | หมายเหตุ |
|--------|------|-----------|----------|
| `id` | VARCHAR(36) | PK, NOT NULL | ตรงกับ user_id ใน auth context |
| `company_id` | VARCHAR(36) | nullable, index | legacy; prefer shop_id |
| `shop_id` | VARCHAR(36) | nullable, index, FK → shops.id | **Tenant scope** — null เฉพาะ Root |
| `email` | VARCHAR(256) | unique (where deleted_at IS NULL) | 1 user : 1 shop |
| `password_hash` | VARCHAR(256) | nullable | bcrypt; สำหรับ login |
| `display_name` | VARCHAR(256) | nullable | |
| `role` | VARCHAR(32) | NOT NULL | Root \| SuperAdmin \| Admin \| Affiliate |
| `tier` | VARCHAR(16) | NOT NULL | free / paid |
| `tier_started_at` | TIMESTAMPTZ | nullable | เริ่มต้น tier ปัจจุบัน (จาก invite) |
| `tier_expires_at` | TIMESTAMPTZ | nullable | หมดอายุ (null = unlimited) |
| `invite_code_used` | VARCHAR(36) | nullable | invite code ที่ใช้ล่าสุด |
| `invite_slots` | INT | NOT NULL, default 0 | จำนวนเชิญที่เหลือ (สำหรับ referral) |
| `created_at` | TIMESTAMPTZ | NOT NULL | |
| `updated_at` | TIMESTAMPTZ | NOT NULL | |
| `deleted_at` | TIMESTAMPTZ | nullable | soft delete |

- **เป็น tenant-scoped (shop):** ทุก query/update ที่เกี่ยวกับข้อมูลร้านต้อง filter ตาม `shop_id` ที่ได้จาก **auth context** เท่านั้น. Root มี shop_id = null.

### 2.3 InviteCode (tier management)

| ฟิลด์ | Type | ข้อกำหนด | หมายเหตุ |
|--------|------|-----------|----------|
| `id` | VARCHAR(36) | PK, NOT NULL | UUID |
| `code` | VARCHAR(64) | unique, NOT NULL | "STOCK-XXXXXX" (6 chars A-Z0-9) |
| `grant_tier` | VARCHAR(16) | NOT NULL | "free" / "paid" ที่จะให้เมื่อใช้ code |
| `tier_duration_days` | INT | nullable | จำนวนวัน (null = unlimited) |
| `max_uses` | INT | nullable | จำนวนครั้งใช้ได้สูงสุด (null = unlimited) |
| `used_count` | INT | NOT NULL, default 0 | นับจำนวนครั้งที่ใช้แล้ว |
| `is_active` | BOOLEAN | NOT NULL, default true | สถานะ (ปิด/เปิด โดย admin) |
| `expires_at` | TIMESTAMPTZ | nullable | หมดอายุ code เอง (null = no expiry) |
| `description` | TEXT | nullable | หมายเหตุจาก admin |
| `created_at` | TIMESTAMPTZ | NOT NULL | |
| `updated_at` | TIMESTAMPTZ | NOT NULL | |

- **ไม่ tenant-scoped** — invite code ใช้ได้ทั่วระบบ (Root/SuperAdmin สร้าง).

### 2.4 TierHistory (audit log)

| ฟิลด์ | Type | ข้อกำหนด | หมายเหตุ |
|--------|------|-----------|----------|
| `id` | BIGSERIAL | PK, NOT NULL | auto-increment |
| `user_id` | VARCHAR(36) | NOT NULL, FK → users.id | ผู้ใช้ที่เปลี่ยน tier |
| `old_tier` | VARCHAR(16) | NOT NULL | tier ก่อนเปลี่ยน |
| `new_tier` | VARCHAR(16) | NOT NULL | tier หลังเปลี่ยน |
| `reason` | VARCHAR(64) | NOT NULL | "invite_code" / "admin_grant" / "expired" |
| `invite_code_id` | VARCHAR(36) | nullable, FK → invite_codes.id | ถ้าเปลี่ยนจาก code |
| `started_at` | TIMESTAMPTZ | NOT NULL | วันที่เริ่ม tier ใหม่ |
| `expires_at` | TIMESTAMPTZ | nullable | วันหมดอายุ (null = unlimited) |
| `created_at` | TIMESTAMPTZ | NOT NULL | เวลาที่บันทึก |

- **tenant-scoped (ผ่าน user)** — filter ตาม user.shop_id เมื่อ query history.

### 2.5 SystemConfig (global settings)

| ฟิลด์ | Type | ข้อกำหนด | หมายเหตุ |
|--------|------|-----------|----------|
| `id` | BIGSERIAL | PK, NOT NULL | auto-increment |
| `key` | VARCHAR(128) | unique, NOT NULL | เช่น "require_invite_code" |
| `value` | TEXT | NOT NULL | JSON string หรือ plain text |
| `description` | TEXT | nullable | อธิบาย setting |
| `created_at` | TIMESTAMPTZ | NOT NULL | |
| `updated_at` | TIMESTAMPTZ | NOT NULL | |

- **ไม่ tenant-scoped** — global config (Root only).

### 2.6 Company (legacy)

- ตาราง `companies` ยังมีอยู่ได้สำหรับ legacy; ใช้ `shops` เป็น tenant หลัก.

---

## 3. ER (ความสัมพันธ์)

```
┌─────────────────┐         ┌─────────────────────────┐         ┌────────────────┐
│     shops       │         │        users            │         │ invite_codes   │
├─────────────────┤         ├─────────────────────────┤         ├────────────────┤
│ id (PK)         │◄────────│ shop_id (FK)            │         │ id (PK)        │
│ name            │    1  * │ id (PK)                 │         │ code (unique)  │
│ created_at      │         │ email (unique)          │         │ grant_tier     │
│ updated_at      │         │ password_hash           │         │ max_uses       │
│ deleted_at      │         │ display_name            │         │ used_count     │
└─────────────────┘         │ role, tier              │         │ is_active      │
                             │ tier_started_at         │         │ expires_at     │
                             │ tier_expires_at         │         └────────────────┘
                             │ invite_code_used        │──────┐          │
                             │ invite_slots            │      │          │
                             │ created_at, updated_at  │      │          │
                             │ deleted_at              │      │          │
                             └─────────────────────────┘      │          │
                                      │                        │          │
                                      │ 1                      │          │
                                      │                        │          │
                                      │ *                      │          │
                                      ▼                        ▼          │
                             ┌────────────────────┐    ┌──────▼──────────▼─┐
                             │   tier_history     │    │  system_config    │
                             ├────────────────────┤    ├───────────────────┤
                             │ id (PK)            │    │ id (PK)           │
                             │ user_id (FK)       │    │ key (unique)      │
                             │ old_tier→new_tier  │    │ value             │
                             │ reason             │    │ description       │
                             │ invite_code_id (FK)│    │ created_at        │
                             │ started_at         │    │ updated_at        │
                             │ expires_at         │    └───────────────────┘
                             │ created_at         │
                             └────────────────────┘
```

### ความสัมพันธ์

- **Shop 1 — * User:** User หนึ่งคนสังกัดร้านเดียว (`users.shop_id` → `shops.id`). Root มี shop_id = null.
- **User 1 — * TierHistory:** ประวัติการเปลี่ยน tier ของแต่ละ user (`tier_history.user_id` → `users.id`).
- **InviteCode 1 — * TierHistory:** เมื่อใช้ invite code ให้สร้าง tier_history (`tier_history.invite_code_id` → `invite_codes.id`).
- **SystemConfig:** ไม่มี FK — เป็นตาราง config แบบ key-value (เช่น `require_invite_code = "true"`).
- ตารางอื่นที่แยกตามร้านในอนาคต (เช่น inventory, orders) ต้องมี `shop_id` และ scope ตาม auth context.

---

## 4. Tenant injection (กฎการใส่ / ใช้ shop_id)

- **ที่มา shop_id ฝั่ง backend:** มาจาก **auth context เท่านั้น** (ที่ middleware ดึงจาก JWT/session หลัง login). ไม่ใช้ค่าจาก request body หรือ query parameter เป็น **scope** ของข้อมูล.
- **การ inject:**  
  - ใน handler ที่เข้าถึงข้อมูล tenant-scoped ให้ใช้ `middleware.GetContext(r.Context()).ShopID` เป็นค่าเดียวที่ใช้ filter / set ใน query หรือ insert/update.
- **สิ่งที่ห้าม:**  
  - ห้ามใช้ `shop_id` จาก JSON body หรือ query string เพื่อกำหนดว่า “ดู/แก้ข้อมูลของเจ้าไหน”.  
  - ห้ามข้าม middleware auth แล้วไปอ่าน shop_id จากที่อื่น. 
---

## 5. ตารางที่ถือว่า tenant-scoped (ปัจจุบันและอนาคต)

| ตาราง | มี shop_id | หมายเหตุ |
|--------|------------|----------|
| `shops` | ไม่มี | เป็นตัว tenant เอง |
| `users` | มี (nullable สำหรับ Root) | scope ตาม shop ที่ user สังกัด |
| `tier_history` | ทาง user | filter ตาม user.shop_id เมื่อ query history |
| `invite_codes` | ไม่มี | ใช้ได้ทั่วระบบ (Root/SuperAdmin สร้าง) |
| `system_config` | ไม่มี | global config (Root only) |
| (อนาคต) inventory, orders, … | มี | ต้องใช้ shop_id จาก auth context เท่านั้น |

---

## 6. Acceptance criteria

| ID | Criteria | วิธีตรวจ |
|----|----------|----------|
| AC1 | ทุก entity ตาม spec มีฟิลด์และ constraint ตรงกับตารางด้านบน | เปรียบเทียบ migration / model กับ spec |
| AC2 | ทุก query/update ที่เป็น tenant-scoped ใช้ shop_id จาก auth context เท่านั้น | Code review; ไม่มีการอ่าน shop_id จาก body/query สำหรับ scope |
| AC3 | มี ER หรือเอกสารความสัมพันธ์ Shop — User | ตรวจ docs/ENTITY_SPEC.md และ migration (000003) |

---

## 7. ข้อควรระวังและกรณีที่อนาคตอาจกระทบ (ต้องเข้มงวดในการ implement)

**สรุปจากวิจัย (OWASP Multi-Tenant, tenant isolation):**

- **ห้ามเชื่อถือ shop_id จาก client** — ใช้จาก auth context เท่านั้น; ใช้จาก body/query/header เป็น scope = tenant context injection.
- **ห้ามลืม WHERE shop_id** — ทุก SELECT/UPDATE/DELETE ที่ tenant-scoped ต้องมีเงื่อนไขจาก auth context ShopID; ลืมแล้วคืนข้อมูลข้ามร้านโดยไม่ error.
- **Tenant context เป็น per-request เท่านั้น** — ไม่อยู่ต่อ connection/global; ถ้ามี background job แต่ละ job ต้องรับ/resolve tenant ของ job (ไม่ใช้ request context).
- **RLS ไม่พอถ้า app ไม่ส่ง context** — App ต้องส่ง shop_id จาก auth context ในทุก query ที่ tenant-scoped.

**กรณีที่อนาคตอาจกระทบ:**

| กรณี | การป้องกัน |
|------|------------|
| เพิ่ม endpoint ใหม่ที่อ่าน/เขียนข้อมูล | ระบุใน feature spec ว่า Auth, Permission, Tenant scope; ใช้ TenantScope(ctx) ใน query |
| เพิ่มตารางที่แยกตามเจ้า | ตารางต้องมี shop_id + index; ทุก query ใช้ค่าจาก context เท่านั้น |
| Background job | แต่ละ job ต้องมี tenant identifier (เช่น จาก queue); resolve shop_id ใน job; ไม่ใช้ request context |
| Query raw / สร้าง query จาก string | Parameterized เท่านั้น; ใส่ shop_id จาก context เป็น parameter |
| Query param สำหรับ filter shop | ห้ามใช้เป็น tenant scope; ใช้เฉพาะจาก auth context |
| Cache แยกตาม tenant | Cache key ต้องรวม shop_id จาก context |

---

## 8. อ้างอิง

- **User context (ความหมาย role, tier, company):** `account-stock-fe/docs/USER_SPEC.md`
- **RBAC (role, permission):** `account-stock-fe/docs/RBAC_SPEC.md`
- **Backend context & API:** this repo `project-specific_context.md`
- **DB & migration:** this repo `docs/DB_SPEC.md`
- **Security (JWT, algorithm, claims):** this repo `docs/SECURITY.md`
