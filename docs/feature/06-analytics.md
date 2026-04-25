# Feature Spec — Analytics

API สำหรับ analytics: reconciliation, daily metrics, product metrics, trends, และ profitability. Role-aware (Affiliate/Seller branch). Date range capped at 365 days.

---

## Routes

| Method | Path | Auth | Permission | Description |
|--------|------|------|------------|-------------|
| GET | `/api/analytics/reconciliation` | Required | `analytics:read` | Reconciliation summary (GMV vs settlement) |
| GET | `/api/analytics/daily-metrics` | Required | `analytics:read` | Time-series daily aggregation |
| GET | `/api/analytics/product-metrics` | Required | `analytics:read` | Per-SKU aggregation |
| GET | `/api/analytics/trends` | Required | `analytics:read` | Trend buckets (weekly/monthly) with growth |
| GET | `/api/analytics/profitability` | Required | `analytics:read` | Margin distribution and category breakdown |

---

## Common Query Parameters

| Param | Type | Default | Validation |
|-------|------|---------|------------|
| `from` | string (YYYY-MM-DD) | today - 29 days | Valid date format |
| `to` | string (YYYY-MM-DD) | today | Valid date format |
| `period` | string | `monthly` | `weekly` or `monthly` (trends only) |

**Date range cap:** Maximum 365 days. If range exceeds, `from` is capped to `to - 365 days`.

**Tenant scope:** ทุก endpoint ใช้ shop_id/company_id จาก auth context. Root ไม่มี data (no shop).

---

## GET /api/analytics/reconciliation

**Response (200):**

```json
{
  "hasData": true,
  "totals": {
    "gmv": 500000.00,
    "settlement": 350000.00,
    "fees": 150000.00,
    "feeRate": 30.0,
    "refund": 10000.00,
    "deductions": 140000.00
  },
  "from": "2026-03-01",
  "to": "2026-03-28"
}
```

**Logic:**
- Seller: gmv = SUM(revenue), settlement = SUM(net), fees = gmv - settlement, refund = SUM(refund), deductions = SUM(deductions)
- Affiliate: gmv = SUM(gmv), settlement = SUM(commission_amount), fees = SUM(ineligible_amount)
- feeRate = (fees / gmv) * 100 (ถ้า gmv = 0 → feeRate = 0)

---

## GET /api/analytics/daily-metrics

**Response (200):**

```json
{
  "hasData": true,
  "totals": {
    "revenue": 500000.00,
    "profit": 350000.00,
    "settlement": 350000.00
  },
  "timeSeries": [
    {"label": "2026-03-28", "revenue": 15000, "profit": 10500, "settlement": 10500},
    {"label": "2026-03-27", "revenue": 12000, "profit": 8400, "settlement": 8400}
  ],
  "from": "2026-03-01",
  "to": "2026-03-28"
}
```

**Logic:**
- Group by date (YYYY-MM-DD)
- Fill missing dates with zeros
- Seller: revenue = SUM(revenue), profit = SUM(net), settlement = SUM(net)
- Affiliate: revenue = SUM(gmv), profit = SUM(commission_amount)

---

## GET /api/analytics/product-metrics

**Response (200):**

```json
{
  "hasData": true,
  "products": [
    {
      "skuId": "ABC123",
      "name": "เสื้อยืด",
      "category": "general",
      "quantity": 200,
      "revenue": 50000.00,
      "profit": 35000.00,
      "profitMargin": 70.0,
      "hasCost": false
    }
  ],
  "from": "2026-03-01",
  "to": "2026-03-28"
}
```

**Logic:**
- Group by sku_id
- margin = (profit / revenue) * 100 (ถ้า revenue = 0 → margin = null)
- Sorted by revenue DESC (implied by Go map iteration + sort)
- hasCost: false (ยังไม่มี cost data ใน current schema)

---

## GET /api/analytics/trends

**Query params:** `period` = `weekly` | `monthly`

**Response (200):**

```json
{
  "hasData": true,
  "period": "monthly",
  "buckets": [
    {"label": "2026-03", "revenue": 100000, "profit": 70000, "count": 300}
  ],
  "monthlyBuckets": [
    {"label": "2026-03", "revenue": 100000, "profit": 70000, "count": 300}
  ],
  "momGrowth": [
    {"label": "2026-03", "revenueGrowth": 15.5, "profitGrowth": 12.3}
  ],
  "yoy": [
    {"label": "2026-03", "revenueGrowth": null, "profitGrowth": null}
  ],
  "from": "2026-03-01",
  "to": "2026-03-28"
}
```

**Logic:**
- Weekly: group by ISO week (YYYY-WNN)
- Monthly: group by year-month (YYYY-MM)
- MoM growth: (current - previous) / previous * 100
- YoY: compare same month previous year (null if no data)

---

## GET /api/analytics/profitability

**Response (200):**

```json
{
  "hasData": true,
  "avgMargin": 25.5,
  "marginBuckets": [
    {"range": "<0%", "count": 5},
    {"range": "0-15%", "count": 20},
    {"range": "15-25%", "count": 50},
    {"range": "25%+", "count": 75}
  ],
  "byCategory": [
    {"category": "general", "profit": 350000, "margin": 25.5}
  ],
  "marginTrend": [
    {"label": "2026-01", "margin": 22.3},
    {"label": "2026-02", "margin": 24.1},
    {"label": "2026-03", "margin": 25.5}
  ],
  "from": "2026-03-01",
  "to": "2026-03-28"
}
```

**Logic:**
- avgMargin: ค่าเฉลี่ย margin ของทุก SKU (เฉพาะ SKU ที่ revenue > 0)
- marginBuckets: แบ่ง margin เป็น 4 กลุ่ม (<0%, 0-15%, 15-25%, 25%+)
- byCategory: ปัจจุบัน "general" เท่านั้น (ยังไม่มี category ใน schema)
- marginTrend: aggregate margin by month (avg of daily margins)
- ถ้า revenue = 0 → ข้าม SKU นั้น (ไม่นับในค่าเฉลี่ย)

---

## Error Responses

| Status | Message | Condition |
|--------|---------|-----------|
| 401 | `unauthorized` | Missing/invalid JWT |
| 403 | `forbidden` | No `analytics:read` permission |
| 500 | `internal error` | DB error |

ทุก error response ใช้ WriteJSONError กับ fixed messages (ห้ามใส่ user input ใน error body).

---

## Acceptance Criteria

1. ทุก endpoint ต้องมี Auth + RequirePermission(`analytics:read`) middleware
2. Tenant scope: query ต้อง filter ด้วย shop_id/company_id จาก context เสมอ
3. Affiliate role ต้อง query จาก `affiliate_sku_row`, ไม่ใช่ `import_sku_row`
4. Date range cap: ห้ามเกิน 365 วัน (auto-cap `from` ถ้าเกิน)
5. Division by zero: ต้องตรวจทุกที่ (revenue=0 → margin=null/0, gmv=0 → feeRate=0)
6. Daily metrics ต้อง fill missing dates ด้วย 0
7. Trends ต้อง support ทั้ง weekly และ monthly period
8. Response เป็น JSON, Content-Type: application/json
9. Error responses ใช้ fixed messages เท่านั้น (OWASP A03)
