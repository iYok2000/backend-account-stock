# Feature Spec — Dashboard

API สำหรับ dashboard overview, KPIs, revenue chart, และ low-stock alerts. Role-aware (Affiliate ดูข้อมูล affiliate_sku_row, Seller/Admin ดูข้อมูล import_sku_row).

---

## Routes

| Method | Path | Auth | Permission | Description |
|--------|------|------|------------|-------------|
| GET | `/api/dashboard/overview` | Required | `dashboard:read` | Overview stats (total products, revenue, low stock) |
| GET | `/api/dashboard/kpis` | Required | `dashboard:read` | KPI totals (revenue, deductions, net) |
| GET | `/api/dashboard/revenue-7d` | Required | `dashboard:read` | Revenue time-series (last 7 days) |
| GET | `/api/dashboard/low-stock` | Required | `dashboard:read` | SKUs with low quantity |

---

## GET /api/dashboard/overview

**Tenant scope:** shop_id จาก auth context (ถ้ามี), หรือ company_id → lookup shop IDs. Root: ไม่มี data (no shop).

**Response (200):**

```json
{
  "total_products": 150,
  "total_revenue": 500000.00,
  "low_stock_count": 12,
  "last_import_date": "2026-03-28T10:00:00Z"
}
```

**Logic:**
- `total_products`: COUNT DISTINCT sku_id
- `low_stock_count`: จำนวน SKU ที่ latest quantity < 5 (threshold = 5)
- `last_import_date`: MAX(date) จาก import_sku_row หรือ affiliate_sku_row
- Affiliate role: query `affiliate_sku_row`, ใช้ commission_amount แทน revenue
- Seller/Admin: query `import_sku_row`

**Error:** 401 (no token), 403 (no dashboard:read)

---

## GET /api/dashboard/kpis

**Query params:** ไม่มี (aggregate ทุก data ใน scope)

**Tenant scope:** เหมือน overview

**Response (200):**

```json
{
  "total_revenue": 500000.00,
  "total_deductions": 75000.00,
  "total_net": 350000.00
}
```

**Logic:**
- Affiliate: revenue = SUM(gmv), deductions = SUM(ineligible_amount), net = SUM(commission_amount)
- Seller: revenue = SUM(revenue), deductions = SUM(deductions), net = SUM(net)

---

## GET /api/dashboard/revenue-7d

**Response (200):**

```json
{
  "days": [
    {"date": "2026-03-28", "amount": 15000.00, "units": 45},
    {"date": "2026-03-27", "amount": 12000.00, "units": 38}
  ]
}
```

**Logic:**
- คำนวณ 7 วันล่าสุด (Bangkok timezone)
- วันที่ไม่มี data → fill ด้วย `{amount: 0, units: 0}`
- Affiliate: amount = SUM(commission_amount), units = SUM(items_sold)
- Seller: amount = SUM(revenue), units = SUM(quantity)

---

## GET /api/dashboard/low-stock

**Query params:**
| Param | Type | Default | Validation |
|-------|------|---------|------------|
| `limit` | int | 5 | 1-100 |

**Response (200):**

```json
{
  "items": [
    {"sku_id": "ABC123", "product_name": "เสื้อยืด", "quantity": 3, "revenue": 1500.00}
  ],
  "threshold": 5
}
```

**Logic:**
- ดึง SKU ที่ latest record มี quantity < threshold
- เรียงตาม quantity ASC (ต่ำสุดก่อน)
- Limit ตาม param (default 5)
- Affiliate: ใช้ items_sold แทน quantity, commission_amount แทน revenue

---

## Acceptance Criteria

1. ทุก endpoint ต้องมี Auth + RequirePermission(`dashboard:read`) middleware
2. Tenant scope: query ต้อง filter ด้วย shop_id หรือ company_id จาก context เสมอ
3. Affiliate role ต้อง query จาก `affiliate_sku_row`, ไม่ใช่ `import_sku_row`
4. Response เป็น JSON, Content-Type: application/json
5. Error responses ใช้ WriteJSONError กับ fixed messages เท่านั้น
6. revenue-7d ต้อง fill missing days ด้วย 0 (ไม่มีช่องว่างใน time-series)
7. low-stock limit ต้อง validate range (1-100, default 5)
