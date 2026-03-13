# DB Spec — GORM + PostgreSQL / Supabase

Spec สำหรับ layer ฐานข้อมูลของ **account-stock-be** ให้สอดคล้องกับ USER_SPEC, RBAC_SPEC และ project-specific_context (multi-tenant, company_id scope).

---

## 1. Scope

- **ORM:** GORM (`gorm.io/gorm`).
- **Driver:** PostgreSQL (`gorm.io/driver/postgres`) — ใช้ได้ทั้ง **PostgreSQL** (self-hosted / RDS) และ **Supabase** (managed Postgres); ใช้ driver เดียวกัน ต่างกันแค่ DSN.
- **Config:** อ่านจาก env — รองรับทั้งคู่:
  - **PostgreSQL:** `DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable` (หรือแยก host/port/user/password/dbname).
  - **Supabase:** `DATABASE_URL` หรือ `SUPABASE_DB_URL` จาก Supabase Dashboard → Project Settings → Database. ถ้าใช้ Supabase ต้องใส่ `?sslmode=require`. ขั้นตอนละเอียด: **`docs/SUPABASE.md`**.

---

## 2. Multi-tenant (shop_id)

- ตารางที่แยกตามร้าน **ต้องมีคอลัมน์ `shop_id`** (FK → shops.id) และเป็น index ตาม project-specific_context.
- ทุก query / update / delete ที่เป็น tenant-scoped **ต้อง filter ตาม `shop_id` จาก auth context** — ห้ามใช้ค่าจาก client ใน body/query param เป็น scope. Root มี shop_id = null.
- GORM: ใช้ `.Where("company_id = ?", companyID)` หรือ scope helper; **ห้าม** ต่อ string จาก user เข้า SQL (parameterized only ตาม SECURITY.md A03).

---

## 3. โครงสร้างโฟลเดอร์ / package

- **`internal/database`** — เปิด connection, ปิด, config; ไม่ใส่ business logic.
- **`internal/model`** — GORM models เท่านั้น (struct, table name); ตาราง tenant-scoped มีฟิลด์ `ShopID` (และตาราง `shops`). User มี `password_hash`, `shop_id` (nullable).
- **`migrations/`** — SQL migration แบบ versioned (`.up.sql` / `.down.sql`) ใช้กับ Postgres และ Supabase เหมือนกัน; รันผ่าน `go run ./cmd/migrate` หรือ `migrate` CLI.

---

## 3.1 Schema migration

- **รูปแบบ:** Versioned SQL migrations (เลขเวอร์ชัน + `.up.sql` / `.down.sql`) — ใช้ได้ทั้ง Postgres และ Supabase เพราะเป็น Postgres เหมือนกัน.
- **ที่เก็บ:** `migrations/` ใน repo (เช่น `000001_init.up.sql`, `000001_init.down.sql`).
- **การรัน:** `go run ./cmd/migrate` (อ่าน `DATABASE_URL` หรือ `SUPABASE_DB_URL` จาก env แล้วรัน up) หรือใช้ [golang-migrate](https://github.com/golang-migrate/migrate) CLI: `migrate -path migrations -database "$DATABASE_URL" up`.
- **หลักการ:** ไม่แก้ migration ที่รันไปแล้ว; เพิ่มเวอร์ชันใหม่เท่านั้น. Production ต้อง backup ก่อน down.

### 3.2 เมื่อมีข้อมูลแล้ว — เพิ่มหรือลด field (column)

เมื่อตารางมีข้อมูลอยู่แล้ว ห้ามไปแก้ไฟล์เก่า (เช่น `000001_init.up.sql`) เพราะ migration นั้นอาจรันไปแล้วบน production. ให้ทำแบบนี้:

| กรณี | วิธีทำ |
|------|--------|
| **เพิ่ม field** | สร้าง migration ใหม่ เช่น `000002_add_user_phone.up.sql` ใช้ `ALTER TABLE ... ADD COLUMN ...`; ไฟล์ `.down.sql` ใช้ `ALTER TABLE ... DROP COLUMN ...` เพื่อ rollback ได้ |
| **ลด field** | สร้าง migration ใหม่ เช่น `000003_remove_user_phone.up.sql` ใช้ `ALTER TABLE ... DROP COLUMN ...`; ไฟล์ `.down.sql` ใช้ `ALTER TABLE ... ADD COLUMN ...` (กำหนด type/default เดิม) เพื่อ rollback ได้ |

**ลำดับที่ทำทุกครั้ง:**

1. สร้างคู่ไฟล์ `00000N_description.up.sql` และ `00000N_description.down.sql`
2. เขียน SQL ใน `.up.sql` (ADD COLUMN หรือ DROP COLUMN ฯลฯ)
3. เขียน SQL ใน `.down.sql` ให้ย้อนการกระทำใน `.up.sql` (ถ้า up = ADD แล้ว down = DROP; ถ้า up = DROP แล้ว down = ADD กลับมา)
4. อัปเดต GORM model ใน `internal/model` ให้ตรงกับ schema ใหม่
5. รัน `go run ./cmd/migrate` (หรือ `migrate ... up`) — จะรันเฉพาะเวอร์ชันที่ยังไม่รัน

**ข้อควรระวัง:**

- **เพิ่ม column:** ถ้าใส่ `NOT NULL` โดยไม่มี default ต้องใส่ default ในคำสั่ง ADD หรือทำ 2 ขั้น (ADD เป็น nullable → อัปเดตข้อมูล → ALTER SET NOT NULL) เพื่อไม่ให้ row เก่าพัง
- **ลบ column:** ข้อมูลใน column นั้นจะหายไป; ถ้าต้องการเก็บไว้ ให้ย้ายไปที่อื่นหรือ export ก่อน
- **Production:** ก่อนรัน down ต้อง backup; ควรทดสอบ up/down บน copy ของ DB ก่อน

**ตัวอย่างโครงไฟล์ (ลบ column):**

```
000003_remove_user_phone.up.sql   →  ALTER TABLE users DROP COLUMN IF EXISTS phone;
000003_remove_user_phone.down.sql →  ALTER TABLE users ADD COLUMN phone VARCHAR(32);
```

(ถ้าไม่ต้องการตัวอย่าง `000002_add_user_phone` ใน repo สามารถลบ 000002 ออกและใช้เลขเวอร์ชันถัดไปสำหรับ migration จริง)

### 3.3 Migrations ปัจจุบัน

| เวอร์ชัน | ชื่อไฟล์ | คำอธิบาย |
|----------|----------|----------|
| 000001 | `init` | สร้างตาราง users, shops, companies (initial schema) |
| 000002 | `add_user_phone` | เพิ่ม phone column (ตัวอย่าง — สามารถลบได้) |
| 000003 | `shops_and_user_shop` | เพิ่ม shops table และ user.shop_id (multi-tenant) |
| 000005 | `import_results` | สร้างตาราง import_results (order import tracking) |
| 000006 | `import_sku_row` | สร้างตาราง import_sku_rows (SKU-level import data) |
| **000015** | `invite_system` | **เพิ่ม invite_codes, tier_history, system_config; เพิ่มฟิลด์ tier tracking ใน users** |

**Migration 000015 (Invite System):**
- **ตาราง:**
  - `invite_codes` — เก็บ invite codes (STOCK-XXXXXX) สำหรับควบคุม tier access
  - `tier_history` — audit log การเปลี่ยน tier (old→new, reason, invite code used)
  - `system_config` — key-value config (เช่น `require_invite_code`)
- **แก้ไข users:**
  - เพิ่ม `tier_started_at`, `tier_expires_at`, `invite_code_used`, `invite_slots`
- **API:** ดู `docs/feature/06-invites.md` สำหรับ endpoint

---

## 4. Security (สอดคล้อง SECURITY.md)

- **Injection:** ใช้เฉพาะ GORM query builder หรือ parameterized raw (`db.Raw("...", arg1, arg2)`); **ห้าม** สร้าง SQL โดยการต่อ string จาก user input.
- **Secrets:** DSN/password อ่านจาก env เท่านั้น; ไม่ commit `.env` / `.env.local`.
- **Logging:** เปิด GORM debug log ได้เฉพาะใน dev; production ระวังไม่ log query ที่มี PII.

---

## 5. Acceptance criteria

| ID | Criteria | วิธีตรวจ |
|----|----------|----------|
| AC1 | Server ต่อ PostgreSQL ได้เมื่อตั้ง `DATABASE_URL` (หรือ equiv.) | รัน server โดยมี env ถูกต้อง แล้วไม่ error ตอน startup |
| AC2 | ตาราง tenant-scoped มี `shop_id` และ query ใช้ค่าจาก auth context | Code review + ทดสอบว่า handler ใช้ ShopID จาก middleware เท่านั้น |
| AC3 | ไม่มี raw SQL ที่ต่อ string จาก user input | Code review + grep ว่ามีเฉพาะ parameterized / GORM builder |
| AC4 | Config อ่านจาก env; มี `.env.example` ระบุ `DATABASE_URL` และ Supabase | ตรวจ `.env.example` และโค้ดที่เปิด DB |
| AC5 | รัน migration up ได้กับทั้ง Postgres และ Supabase (DSN จาก env) | รัน `go run ./cmd/migrate` หรือ migrate CLI กับทั้งสองแบบ DSN |

---

## 6. Security review checklist (DB layer)

- [ ] DSN จาก env เท่านั้น
- [ ] ทุก tenant-scoped query มี `company_id` จาก context
- [ ] ไม่มี `fmt.Sprintf` / string concat สำหรับ SQL
- [ ] GORM debug log ปิดหรือไม่ log PII ใน production

---

## 7. อ้างอิง

- **User & tenant (context):** `account-stock-fe/docs/USER_SPEC.md`, this repo `project-specific_context.md`
- **User & Tenant (entity, ER, tenant injection):** this repo `docs/ENTITY_SPEC.md`
- **RBAC:** `account-stock-fe/docs/RBAC_SPEC.md`
- **Security:** this repo `docs/SECURITY.md`
