# Production Readiness — account-stock-be

Checklist เพื่อให้ระบบ **พร้อมขึ้น production**. อ้างอิง DEPLOY.md, SECURITY.md, ENTITY_SPEC, project-specific_context.

---

## 1. สิ่งที่ต้องมีก่อนขึ้น prod (บังคับ)

| หัวข้อ | ตรวจจาก | สถานะ |
|--------|---------|--------|
| **APP_ENV=production** | Server ไม่รันถ้า JWT_SECRET เป็นค่า dev | บังคับใน main |
| **JWT_SECRET** | ตั้งค่าใหม่ ไม่ใช้ default | ต้องตั้งบน host |
| **CORS_ORIGIN** | ใส่ origin ของ frontend จริง | ต้องตั้งถ้า FE คนละ origin |
| **Auth ทุก protected route** | ไม่มี endpoint ที่ควรต้อง auth แต่เปิดไว้ | GET /api/auth/me, GET /api/users, POST /api/inventory/import, GET /api/inventory, GET /api/inventory/summary ผ่าน Auth |
| **Permission ตาม RBAC** | แต่ละ route ตรวจ permission | users:read → /api/users; inventory:create/read → /api/inventory/* |
| **Error responses** | ใช้ constant เท่านั้น ไม่ส่ง user input (SECURITY A03) | middleware.WriteJSONError |
| **Tenant (company_id)** | ใช้จาก auth context เท่านั้น ไม่รับจาก client | ENTITY_SPEC §4; handler ใช้ TenantScope(ctx) เมื่อมี query tenant-scoped |

---

## 2. Spec ที่ต้องสอดคล้อง

- **project-specific_context.md** — API contract, ไม่รับ company ใน body, auth + RBAC + multi-tenant
- **docs/ENTITY_SPEC.md** — Entity User/Company, กฎ inject company_id
- **docs/DB_SPEC.md** — ตาราง tenant-scoped มี company_id, query ใช้ค่าจาก context
- **docs/SECURITY.md** — OWASP Top 10, injection prevention
- **docs/feature/*.md** — แต่ละ endpoint ระบุ Auth, Permission, Tenant scope

---

## 3. Endpoints และการป้องกัน

| Method | Path | Auth | Permission | Tenant scope |
|--------|------|------|------------|--------------|
| GET | /health | ไม่มี | — | — |
| GET | /api/auth/me | JWT | — | ไม่ใช้ (คืน company_id ใน response) |
| GET | /api/users | JWT | users:read | ใช้เมื่อ query DB (SuperAdmin/ Admin; scope shop/company จาก context) |
| POST | /api/inventory/import | JWT | inventory:create | บังคับใช้ shop/company จาก context |
| GET | /api/inventory | JWT | inventory:read | บังคับใช้ shop/company จาก context |
| GET | /api/inventory/summary | JWT | inventory:read | บังคับใช้ shop/company จาก context |

---

## 4. สิ่งที่ทำแล้วในโค้ด

- Refuse start เมื่อ APP_ENV=production และ JWT_SECRET เป็น default
- CORS จาก env; default localhost/127.0.0.1; ไม่ส่ง Allow-Credentials:true คู่กับ origin:*
- HTTP server timeouts: ReadHeaderTimeout=10s, ReadTimeout=30s, WriteTimeout=60s, IdleTimeout=120s
- Global body limit middleware: 32 MB ป้องกัน DoS จาก oversized payload
- Auth middleware ใส่ CompanyID ลง context
- RequirePermission สำหรับ /api/users (users:read), /api/inventory/* (inventory:create/read)
- WriteJSONError ใช้ constant ทุกจุด ไม่ส่ง raw db errors ให้ client
- TenantScope(ctx) helper พร้อมใช้; Root user บน /api/users returns empty list (ไม่ query ข้าม tenant)
- Invite code lock: ใช้ `clause.Locking{Strength:"UPDATE"}` (GORM v2 SELECT FOR UPDATE) ป้องกัน race condition
- tx.Commit() error checked ทุกจุด
- Root credentials: ใช้ `crypto/subtle.ConstantTimeCompare` (constant-time) ป้องกัน timing attack
- Performance indexes บน (shop_id, date), (user_id, order_date), invite_codes.code, users.email (migration 000015)

---

## 5. ก่อน deploy จริง

1. ตั้ง env บน host ตาม **docs/DEPLOY.md** § ต้องตั้งก่อน deploy
2. รัน migration บน DB ของ production
3. ตรวจว่า CORS_ORIGIN ชี้ไป frontend จริง (ไม่ใช้ localhost)
4. (Optional) Graceful shutdown, rate limit ตาม DEPLOY § สิ่งที่ยังไม่มี

---

## 6. ข้อควรระวังและกรณีอนาคต

- **Tenant:** ดู **docs/ENTITY_SPEC.md** §7 (ข้อควรระวัง + กรณี endpoint ใหม่, ตาราง, background job, query param, cache).
- **Auth/JWT:** ดู **docs/SECURITY.md** § Pitfalls (algorithm, claim, secret + endpoint ใหม่).

## 7. อ้างอิง

- **Deploy:** docs/DEPLOY.md
- **Security:** docs/SECURITY.md
- **Entity & tenant:** docs/ENTITY_SPEC.md
- **Feature API:** docs/feature/README.md, docs/feature/*.md
