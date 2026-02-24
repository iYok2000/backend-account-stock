# Feature Spec — Users (จัดการผู้ใช้)

API สำหรับจัดการผู้ใช้ระบบ — เฉพาะ **SuperAdmin** (permission `users:read` ขึ้นไป). สอดคล้องกับ ENTITY_SPEC (User entity, tenant injection).

---

## Route & Permission

| Method | Path | Auth | Permission |
|--------|------|------|------------|
| GET | `/api/users` | Required (Bearer JWT) | `users:read` (SuperAdmin เท่านั้น) |

---

## Request

- **Header:** `Authorization: Bearer <JWT>`
- **Body:** ไม่มี
- **Query (อนาคต):** อาจมี filter, page (ยังไม่กำหนด)

---

## Response (200)

```json
{
  "users": [
    {
      "id": "...",
      "company_id": "...",
      "email": "...",
      "display_name": "...",
      "role": "Staff",
      "tier": "free"
    }
  ]
}
```

- รายการ user; ตอนนี้ยัง placeholder (คืน `[]`) จนกว่า handler จะดึงจาก DB.

---

## Tenant scope & injection

- **SuperAdmin:** อาจเห็น users ข้าม company (ตาม RBAC). ถ้า implement แบบ scope ตาม company ให้ใช้ **company_id จาก auth context เท่านั้น** สำหรับ filter (หรือ query param เฉพาะเมื่อ role = SuperAdmin และตรวจที่ backend) — ตาม ENTITY_SPEC §4.
- **ไม่รับ company_id จาก body/query เป็น scope** สำหรับ role อื่น.

---

## Error

- **401:** ไม่มี token / token ไม่ valid
- **403:** มี token แต่ไม่มี permission `users:read`

---

## Acceptance criteria

- [ ] GET /api/users โดยไม่มี token → 401
- [ ] GET /api/users ด้วย token ที่ role ไม่ใช่ SuperAdmin (ไม่มี users:read) → 403
- [ ] GET /api/users ด้วย token SuperAdmin → 200 และ body มี key `users` (array)
- [ ] เมื่อต่อ DB: รายการ user ต้องสอดคล้อง ENTITY_SPEC; ไม่ใช้ company_id จาก client เป็น scope เว้นแต่กำหนดใน spec
