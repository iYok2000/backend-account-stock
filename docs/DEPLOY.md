# Deploy — account-stock-be

Checklist และสิ่งที่ต้องตั้งก่อน deploy production.

---

## พร้อมแล้ว

- **Build:** `go build -o server ./cmd/server` ได้ binary เดียว
- **Config จาก env:** ไม่ hardcode secret; ใช้ `JWT_SECRET`, `DATABASE_URL` / `SUPABASE_DB_URL`, `PORT`
- **PORT:** อ่านจาก env `PORT` (default 8080) — ใช้กับ Railway, Render, Fly.io, Heroku ได้
- **Health:** `GET /health` ไม่ต้อง auth ใช้สำหรับ load balancer / health check
- **Security:** JWT validation, RBAC, error responses แบบไม่ inject; ดู `docs/SECURITY.md`
- **Migration:** รัน `go run ./cmd/migrate` ก่อนหรือหลัง deploy (ตาม flow ของ host)

---

## ต้องตั้งก่อน deploy

| Env | บังคับใน production | หมายเหตุ |
|-----|----------------------|----------|
| **APP_ENV** | แนะนำ | ตั้ง `production` เพื่อบังคับ JWT_SECRET (ถ้าไม่ตั้งหรือใช้ค่า dev → server ไม่รัน) |
| **JWT_SECRET** | ใช่ | ห้ามใช้ค่า default; ใช้ค่าที่สร้างใหม่และเก็บใน secret manager |
| **CORS_ORIGIN** | ใช่ (ถ้า FE คนละ origin) | ใส่ origin ของ frontend จริง (เช่น https://your-fe.vercel.app). ห้ามใช้ localhost ใน production |
| **DATABASE_URL** หรือ **SUPABASE_DB_URL** | ใช่ (ถ้าใช้ DB) | Connection string ของ Postgres/Supabase |
| **PORT** | ไม่ (default 8080) | บาง host (Railway, Render) ส่ง PORT มาให้ |
| JWT_ISSUER, JWT_AUDIENCE | ไม่ | ใส่ได้ถ้าต้องการตรวจใน JWT |

---

## สิ่งที่ยังไม่มี (optional ต่อยอด)

- **Graceful shutdown** — รับ SIGTERM แล้วปิด listener / drain request (ส่วนใหญ่ reverse proxy ปิด connection ให้)
- **TLS ใน process** — ใช้ reverse proxy (nginx, cloud load balancer) terminate TLS แทนก็ได้
- **Rate limit** — ตามความเสี่ยงของบริการ
- **Import auth** — `POST /api/import/order-transaction` ยังไม่ตรวจ JWT; ควรใส่ auth ก่อน production

---

## ตัวอย่าง deploy (แนวทางทั่วไป)

1. **Build:** `go build -o server ./cmd/server` (หรือใน CI)
2. **Migration:** รันบน DB ที่ production ใช้ (ใช้ `DATABASE_URL` / `SUPABASE_DB_URL` ของ production)
3. **ตั้ง env บน host:** JWT_SECRET, DATABASE_URL หรือ SUPABASE_DB_URL, PORT (ถ้า host กำหนด)
4. **รัน:** `./server` (หรือตามที่ host กำหนด)
5. **Health check:** ตั้งให้เรียก `GET /health` เป็นระยะ
