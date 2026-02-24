# ต่อ Supabase

ขั้นตอนให้ backend ใช้ฐานข้อมูล Supabase (PostgreSQL แบบ managed).

---

## 1. สร้างโปรเจกต์ Supabase

1. ไปที่ [supabase.com](https://supabase.com) → Sign in → New project
2. ตั้งชื่อโปรเจกต์ เลือก region และรอสร้างเสร็จ

---

## 2. หา Connection string

1. ใน Dashboard ไปที่ **Project Settings** (ไอคอนฟันเฟือง) → **Database**
2. หา **Connection string**
   - **URI** — ใช้ได้เลย (มีรหัสผ่านอยู่แล้ว) หรือ
   - **Connection pooling** (แนะนำสำหรับ server) — เลือก **Transaction** หรือ **Session** แล้ว copy

รูปแบบโดยประมาณ:

```
postgres://postgres.[project-ref]:[YOUR-PASSWORD]@aws-0-[region].pooler.supabase.com:6543/postgres
```

- **พอร์ต 6543** = connection pooler (เหมาะกับ server, จำกัด connection)
- **พอร์ต 5432** = direct connection (ใช้กับ migration หรือ script)
3. แทน `[YOUR-PASSWORD]` ด้วย Database password (ที่ตั้งตอนสร้างโปรเจกต์ หรือ reset ในหน้า Database)
4. ถ้าใช้จากนอก Supabase ต้องใส่ **`?sslmode=require`** ต่อท้าย (บาง client เพิ่มให้แล้ว)

ตัวอย่างเต็ม:

```
postgres://postgres.abcdefgh:yourpassword@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?sslmode=require
```

---

## 3. ตั้งค่า env

สร้างหรือแก้ `.env` (อย่า commit):

```bash
# Supabase Database (ใช้ค่าจากขั้นตอน 2)
SUPABASE_DB_URL=postgres://postgres.xxxxx:yourpassword@aws-0-xx.pooler.supabase.com:6543/postgres?sslmode=require

# หรือใช้ชื่อนี้ก็ได้ (backend รองรับทั้งสอง)
# DATABASE_URL=postgres://...

# JWT (ถ้าใช้ Supabase Auth ภายหลังค่อย sync)
JWT_SECRET=your-256-bit-secret-here
```

---

## 4. รัน migration (สร้างตาราง)

จาก root ของ repo:

```bash
# ใช้ SUPABASE_DB_URL จาก .env
export $(cat .env | xargs)   # หรือใส่ SUPABASE_DB_URL ใน shell
go run ./cmd/migrate
```

ถ้าสำเร็จจะเห็น `migrate: up done` และใน Supabase → **Table Editor** จะมีตาราง `companies`, `users` (และ `schema_migrations` จาก migrate)

---

## 5. รัน server

```bash
go run ./cmd/server
```

ถ้า set `SUPABASE_DB_URL` (หรือ `DATABASE_URL`) ไว้แล้ว server จะต่อ DB ตอนสตาร์ท; ถ้าไม่ตั้ง จะใช้โหมดไม่ต่อ DB (เช่น dev local ไม่มี Postgres).

---

## หมายเหตุ

- **Pooler (6543):** เหมาะกับ app server; จำกัดจำนวน connection ต่อโปรเจกต์
- **Direct (5432):** ใช้รัน migration หรือ one-off script ได้; บางครั้ง Supabase แยก connection string ระหว่าง “Session” กับ “Transaction” — ใช้แบบใดก็ได้กับ GORM
- **รหัสผ่าน:** เก็บใน env เท่านั้น อย่าใส่ในโค้ดหรือ commit
- **RLS (Row Level Security):** Supabase เปิด RLS ได้บนตาราง; ถ้าใช้ RLS ต้อง set policy ให้ตรงกับ `company_id` หรือ role ที่ app ใช้
