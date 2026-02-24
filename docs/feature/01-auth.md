# Feature Spec — Auth

API สำหรับ authentication และ current user context (ให้ frontend AuthContext / useUserContext ใช้).

---

## Route & Permission

| Method | Path | Auth | Permission |
|--------|------|------|------------|
| GET | `/api/auth/me` | Required (Bearer JWT) | — (ทุก user ที่ล็อกอินเข้าได้) |

---

## Request

- **Header:** `Authorization: Bearer <JWT>`
- **Body:** ไม่มี

---

## Response (200)

```json
{
  "user": { "id": "...", "displayName": "..." },
  "roles": ["SuperAdmin"],
  "permissions": ["dashboard:read", "..."],
  "tier": "free",
  "company_id": "default"
}
```

- ตรงกับ frontend `MeResponse` / AuthContext (ดู project-specific_context.md).

---

## Tenant scope

- **ไม่ใช้ tenant scope** — endpoint นี้คืนข้อมูลของ “user ที่ล็อกอิน” เท่านั้น (จาก JWT). ไม่ query ข้อมูลแยกตาม company_id.

---

## Error

- **401:** ไม่มี token / token หมดอายุ / token ไม่ถูกต้อง

---

## Acceptance criteria

- [ ] GET /api/auth/me ด้วย Bearer ที่ valid คืน 200 และ JSON ตาม shape ด้านบน
- [ ] ค่า roles, permissions มาจาก role ใน JWT (backend derive ตาม RBAC); tier, company_id จาก claims
- [ ] ไม่มี token หรือ token ผิด → 401
