# Feature Spec — Analytics API (BE)

**Scope:** ฝั่ง BE เท่านั้น. UI/route ดู **account-stock-fe/docs/feature/09-analytics.md**

**Last Updated:** 2026-03-12

---

## 1. สรุป

| Method | Path | Handler | Query |
|--------|------|---------|--------|
| GET | `/api/analytics/reconciliation` | AnalyticsReconciliation | `from`, `to` |
| GET | `/api/analytics/daily-metrics` | AnalyticsDailyMetrics | `from`, `to` |
| GET | `/api/analytics/product-metrics` | AnalyticsProductMetrics | `from`, `to` |
| GET | `/api/analytics/trends` | AnalyticsTrends | `from`, `to`, `period` |
| GET | `/api/analytics/profitability` | AnalyticsProfitability | `from`, `to` |

- **Auth:** JWT + permission `analytics:read` + Tenant scope (shop_id / affiliate จาก context)
- **ที่มา:** `internal/handler/analytics.go`, route ลงทะเบียนใน `cmd/server/main.go`
- **ข้อมูล:** Owner/Admin/Root ใช้ `import_sku_row` (กรอง `deleted_at IS NULL`, `date BETWEEN from AND to`, `shop_id IN (context)`); Affiliate ใช้ `affiliate_sku_row` (company_id, user_id)

---

## 2. Query ร่วม

ทุก endpoint รับ query `from`, `to` (YYYY-MM-DD). ค่า default ถ้าไม่ส่ง: `to = วันนี้`, `from = to - 29 วัน`. ถ้า `end < start` จะสลับให้.

---

## 3. GET /api/analytics/reconciliation

- **Response:** `gmv`, `settlement`, `totalFees`, `netProfit`, `settlementRate`, `feeBreakdown` (array of `{ label, value }`), `from`, `to`
- **Logic:** sum revenue → GMV, net → settlement; totalFees = GMV - settlement; feeBreakdown = tiktokCommission, deductions, refund (กรองค่าที่เป็น 0 ออก)

---

## 4. GET /api/analytics/daily-metrics

- **Response:** `hasData`, `totals` (`revenue`, `profit`, `settlement`), `timeSeries` (array of `{ label, revenue, profit, settlement }` ต่อวัน), `from`, `to`
- **Logic:** group by date; แต่ละวันมี label = YYYY-MM-DD; วันที่ไม่มีข้อมูลส่ง 0

---

## 5. GET /api/analytics/product-metrics

- **Response:** `products` (array of `{ skuId, name, category, quantity, revenue, profit, profitMargin?, hasCost }`), `hasData`, `from`, `to`
- **Logic:** group by sku_id จาก import_sku_row; profitMargin = (profit/revenue)*100 เมื่อ revenue > 0; category ตอนนี้เป็น "general"; hasCost ฝั่ง BE ส่ง false (FE คำนวณจาก inventory/ต้นทุนที่ใส่)

---

## 6. GET /api/analytics/trends

- **Query:** `from`, `to`, `period` (optional, default `monthly`) = `weekly` | `monthly`
- **Response:**
  - `buckets`: array of `{ key, label, revenue, orders, profit, discount, days }` — bucket ตาม period (สัปดาห์หรือเดือน)
  - `monthlyBuckets`: เหมือน buckets แต่บังคับเป็นรายเดือน
  - `momGrowth`: array of `{ label, growth }` — growth % เทียบเดือนก่อน (revenue)
  - `yoy`: array of `{ label, currentYear, previousYear }` — รายเดือนเปรียบเทียบปีนี้กับปีก่อน
  - `hasData`, `from`, `to`, `period`
- **Logic:**
  - อ่านจาก `import_sku_row` (หรือ affiliate path แปลงเป็น importRow shape)
  - Bucket ตามสัปดาห์ (ISO week key เช่น 2025-W12) หรือเดือน (YYYY-MM)
  - MoM: (revenue ปัจจุบัน - revenue เดือนก่อน) / เดือนก่อน * 100
  - YoY: group ตามเดือน แล้วเปรียบเทียบปีปัจจุบันกับปีก่อน; label = "MM-YYYY"
  - `orders` ตอนนี้ส่ง 0 (ไม่มีข้อมูล order count แยก)

---

## 7. GET /api/analytics/profitability

- **Query:** `from`, `to`
- **Response:**
  - `avgMargin`: margin เฉลี่ยต่อ SKU (profit/revenue*100) เฉพาะ SKU ที่ revenue > 0
  - `marginBuckets`: array of `{ range, count }` — ช่วง &lt;0%, 0-15%, 15-25%, 25%+
  - `byCategory`: array of `{ category, profit, margin }` — ตอนนี้มีแค่ "general" หนึ่งแถว
  - `marginTrend`: array of `{ label, margin }` — margin เฉลี่ยต่อเดือน (label = YYYY-MM)
  - `hasData`, `from`, `to`
- **Logic:** ใช้ skuMap จาก product-metrics; histogram margin ตามช่วง; marginTrend = group by month จาก import rows แล้วเฉลี่ย margin ต่อเดือน

---

## 8. Error / Auth

- **401:** ไม่มีหรือ token ไม่ถูกต้อง → JSON `{ "error": "..." }`
- **405:** method ไม่ใช่ GET
- **500:** DB error หรือไม่ initialize → JSON error
- Tenant: ไม่ใช้ `shop_id` จาก query/body — ใช้จาก auth context เท่านั้น

---

## 9. อ้างอิง

- FE spec: account-stock-fe/docs/feature/09-analytics.md
- Entity / tenant: docs/ENTITY_SPEC.md
- RBAC: PermAnalyticsRead ใน internal/rbac/rbac.go
