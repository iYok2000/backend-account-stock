# Feature Spec — Invite System (Tier Management)

API สำหรับจัดการ **Invite Codes** และ **Tier** (free/paid) ของผู้ใช้. สอดคล้องกับ ENTITY_SPEC § 2.3-2.5 และ USER_SPEC (tier fields). Root/SuperAdmin สามารถสร้าง invite codes; user ใช้ code เพื่อเปลี่ยน tier.

---

## Routes

| Method | Path | Auth | Permission |
|--------|------|------|------------|
| **Public** | | | |
| POST | `/api/invite/validate` | ไม่ต้อง | — (ตรวจ invite code ก่อน register) |
| GET | `/api/invite/check-required` | ไม่ต้อง | — (ดูว่าระบบเปิดใช้ invite หรือไม่) |
| **Auth** | | | |
| POST | `/api/invite/use` | Bearer JWT | — (ใช้ invite code → เปลี่ยน tier) |
| **Admin** | | | |
| GET | `/api/admin/invites` | Bearer JWT | `invites:read` (Root/SuperAdmin) |
| POST | `/api/admin/invites` | Bearer JWT | `invites:create` (Root/SuperAdmin) |
| PUT | `/api/admin/invites/:id` | Bearer JWT | `invites:update` (Root/SuperAdmin) |
| DELETE | `/api/admin/invites/:id` | Bearer JWT | `invites:delete` (Root/SuperAdmin) |
| **Config** | | | |
| GET | `/api/admin/system-config` | Bearer JWT | `config:read` (Root only) |
| PUT | `/api/admin/system-config` | Bearer JWT | `config:update` (Root only) |

---

## POST /api/invite/validate (Public)

**Request (application/json):** `{ "code": "STOCK-ABCDEF" }`

**Response (200):**

```json
{
  "valid": true,
  "code": "STOCK-ABCDEF",
  "grant_tier": "paid",
  "tier_duration_days": 365,
  "remaining_uses": 10,
  "expires_at": "2027-03-13T00:00:00Z"
}
```

**Error:**
- 400 — `code` missing
- 404 — code ไม่มี หรือหมดอายุ หรือใช้จนครบแล้ว
- 200 — `{ "valid": false, "message": "..." }` ถ้า code ไม่ active

**Logic:** ตรวจ invite_codes ว่า code ตรง, is_active=true, expires_at ไม่เลยวันนี้, used_count < max_uses (หรือ max_uses=null)

---

## GET /api/invite/check-required (Public)

**Response (200):**

```json
{ "required": true }
```

**Logic:** อ่าน system_config key=`require_invite_code` → คืน `{ "required": <bool> }`

---

## POST /api/invite/use (Auth)

**Header:** `Authorization: Bearer <JWT>`

**Request (application/json):** `{ "code": "STOCK-ABCDEF" }`

**Response (200):**

```json
{
  "message": "Invite code applied successfully",
  "new_tier": "paid",
  "tier_started_at": "2026-03-13T...",
  "tier_expires_at": "2027-03-13T..."  // หรือ null ถ้า unlimited
}
```

**Error:**
- 400 — `code` missing
- 404 — code ไม่มี หรือหมดอายุ หรือใช้จนครบแล้ว
- 409 — user ใช้ code นี้ไปแล้ว (invite_code_used = code)

**Logic:**
1. ดึง user_id จาก JWT context
2. ตรวจ invite_codes (ตาม validate)
3. **Transaction with row lock** (`FOR UPDATE` บน invite_codes + users)
4. ตรวจว่า user ใช้ code นี้แล้วหรือไม่ (invite_code_used)
5. อัปเดต users: `tier` = grant_tier, `tier_started_at` = NOW(), `tier_expires_at` = NOW() + duration (หรือ null), `invite_code_used` = code
6. เพิ่ม used_count (invite_codes)
7. สร้าง tier_history: old_tier → new_tier, reason="invite_code", invite_code_id, started_at, expires_at
8. Commit

---

## GET /api/admin/invites (Admin)

**Header:** `Authorization: Bearer <JWT>`

**Response (200):**

```json
{
  "invites": [
    {
      "id": "uuid",
      "code": "STOCK-ABCDEF",
      "grant_tier": "paid",
      "tier_duration_days": 365,
      "max_uses": 100,
      "used_count": 23,
      "is_active": true,
      "expires_at": "2027-12-31T...",
      "description": "New Year Promo",
      "created_at": "...",
      "updated_at": "..."
    }
  ]
}
```

**Logic:** ดึงทุก rows จาก invite_codes (ไม่ tenant-scoped — Root/SuperAdmin เห็นทั้งหมด)

---

## POST /api/admin/invites (Admin)

**Header:** `Authorization: Bearer <JWT>`

**Request (application/json):**

```json
{
  "code": "STOCK-XYZ123",  // optional — ถ้าไม่ส่งจะ auto-gen
  "grant_tier": "paid",
  "tier_duration_days": 365,  // nullable — null = unlimited
  "max_uses": 100,  // nullable — null = unlimited
  "expires_at": "2027-12-31T23:59:59Z",  // nullable
  "description": "Campaign ABC"
}
```

**Response (201):**

```json
{
  "id": "uuid",
  "code": "STOCK-XYZ123",
  "grant_tier": "paid",
  "tier_duration_days": 365,
  "max_uses": 100,
  "used_count": 0,
  "is_active": true,
  "expires_at": "2027-12-31...",
  "description": "Campaign ABC",
  "created_at": "...",
  "updated_at": "..."
}
```

**Error:**
- 400 — field validation ล้มเหลว (grant_tier ไม่ใช่ "free"/"paid", code format ผิด)
- 409 — code ซ้ำ (unique constraint)

**Logic:**
- ถ้า code ไม่ส่ง → auto-generate "STOCK-" + 6 chars (A-Z0-9 random)
- ถ้าส่งมา → ตรวจว่าซ้ำไหม (unique)
- is_active = true, used_count = 0
- บันทึกลง invite_codes

---

## PUT /api/admin/invites/:id (Admin)

**Header:** `Authorization: Bearer <JWT>`

**Request (application/json):**

```json
{
  "is_active": false,  // toggle on/off
  "expires_at": "2027-06-30T23:59:59Z",
  "description": "Updated description"
}
```

**Response (200):**

```json
{
  "id": "uuid",
  "code": "STOCK-XYZ123",
  "is_active": false,
  "expires_at": "2027-06-30...",
  "description": "Updated description",
  "updated_at": "..."
}
```

**Error:**
- 404 — id ไม่มี
- 400 — field ผิด

**Logic:** อัปเดต invite_codes WHERE id = :id (ไม่อนุญาตแก้ code, grant_tier, max_uses, used_count)

---

## DELETE /api/admin/invites/:id (Admin)

**Header:** `Authorization: Bearer <JWT>`

**Response (200):**

```json
{ "message": "Invite code deactivated" }
```

**Error:** 404 — id ไม่มี

**Logic:** **ไม่ลบจริง** — ตั้ง `is_active = false` เพื่อป้องกัน FK constraint จาก tier_history

---

## GET /api/admin/system-config (Root)

**Header:** `Authorization: Bearer <JWT>`

**Response (200):**

```json
{
  "require_invite_code": "true"
}
```

**Logic:** ดึงทุก rows จาก system_config → map key:value

---

## PUT /api/admin/system-config (Root)

**Header:** `Authorization: Bearer <JWT>`

**Request (application/json):**

```json
{
  "require_invite_code": "false"
}
```

**Response (200):**

```json
{ "message": "System config updated" }
```

**Logic:** ต่อแต่ละ key ใน body → UPDATE system_config SET value=? WHERE key=?; ถ้ายังไม่มี → INSERT

---

## Tenant Scope

- **invite_codes, system_config:** **ไม่ tenant-scoped** — ใช้ได้ทั่วระบบ (Root/SuperAdmin สร้าง)
- **tier_history:** tenant-scoped ผ่าน user — filter ตาม user.shop_id เมื่อ query history
- **/api/invite/use:** อัปเดต user ตาม user_id จาก JWT context

---

## Acceptance Criteria

| ID | Criteria | วิธีตรวจ |
|----|----------|----------|
| AC1 | `/api/invite/validate` คืน 200 + valid=true ถ้า code ใช้ได้; 404/200+valid=false ถ้าไม่ใช้ได้ | Test curl/Postman กับ code ที่ active/expired/ใช้จนหมด |
| AC2 | `/api/invite/use` เปลี่ยน user.tier และบันทึก tier_history ด้วย transaction | ตรวจ DB หลังเรียก API (ต้องมี tier_history row ใหม่) |
| AC3 | `/api/invite/use` ล็อค row (FOR UPDATE) ป้องกัน race condition | Test concurrent requests กับ code เดียวกัน (max_uses ต้องไม่เกิน) |
| AC4 | `/api/admin/invites` (POST) auto-generate code ถ้าไม่ส่งมา | เรียกโดยไม่ส่ง `code` → ได้ "STOCK-XXXXXX" |
| AC5 | `/api/admin/invites/:id` (DELETE) ตั้ง is_active=false แทนลบ | Query DB หลัง DELETE → is_active=false แต่ row ยังมีอยู่ |
| AC6 | `/api/admin/system-config` (PUT) Root only | Non-Root เรียก → 403 |

---

## Security Notes

- **Validate input:** code format (max 64 chars), grant_tier enum, expires_at วันที่ valid
- **Transaction + Row Lock:** `/api/invite/use` ต้องใช้ transaction + `FOR UPDATE` ป้องกัน race condition (max_uses exceeded)
- **Permission check:** middleware ต้องตรวจ `invites:*` และ `config:*` ตาม RBAC (Root/SuperAdmin)
- **ไม่ log code ใน production logs** — ป้องกัน code รั่วไหล

---

## Frontend Integration

- **หน้า Admin/Invites:** `/admin/invites` แสดงตาราง invite codes + สร้าง/แก้/ลบ
- **Landing/Register flow:** ตรวจ `/api/invite/check-required` → ถ้า true แสดง input code → validate → register
- **User tier display:** แสดง user.tier, tier_started_at, tier_expires_at ในหน้า Settings/Profile

---

## อ้างอิง

- **Entity:** this repo `docs/ENTITY_SPEC.md` § 2.3-2.5
- **Migration:** this repo `migrations/000015_invite_system.up.sql`
- **User context:** `account-stock-fe/docs/USER_SPEC.md` (tier fields)
- **RBAC:** `account-stock-fe/docs/RBAC_SPEC.md` (invites:*, config:* permissions)
