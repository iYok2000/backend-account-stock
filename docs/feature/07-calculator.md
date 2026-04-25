# Feature Spec — Calculator (Fee & Profit)

API สำหรับคำนวณค่าธรรมเนียม TikTok Shop และกำไร เพื่อให้ผู้ใช้ประเมินต้นทุน/กำไรก่อนขาย

---

## Routes

| Method | Path | Auth | Permission |
|--------|------|------|------------|
| POST | `/api/calculator/fees` | Bearer JWT | `analytics:read` |
| POST | `/api/calculator/fees/batch` | Bearer JWT | `analytics:read` |

---

## POST /api/calculator/fees

คำนวณค่าธรรมเนียม TikTok Shop สำหรับราคาขายที่กำหนด

**Request (application/json)**

```json
{
  "salePrice": 1000,
  "category": "beauty",
  "overrides": {
    "commissionRate": 0.064,
    "enableLiveSpecials": true,
    "liveSpecialsRate": 0.027
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `salePrice` | number | Yes | ราคาขายหลังหักส่วนลด (บาท) |
| `category` | string | No | หมวดหมู่สินค้า (ใช้ default rate ถ้าไม่ระบุ) |
| `overrides` | object | No | ปรับอัตราค่าธรรมเนียมแบบกำหนดเอง |

**Response (200)**

```json
{
  "salePrice": 1000,
  "category": "beauty",
  "fees": {
    "commission": "74.90",
    "commissionRate": "7.49",
    "paymentFee": "32.10",
    "commerceGrowthFee": "0.00",
    "infrastructureFee": "0.00",
    "liveSpecialsFee": "0.00",
    "eamsFee": "0.00",
    "preorderFee": "0.00",
    "totalFees": "107.00",
    "effectiveRate": "10.70"
  }
}
```

---

## POST /api/calculator/fees/batch

คำนวณค่าธรรมเนียมสำหรับหลายรายการพร้อมกัน

**Request (application/json)**

```json
{
  "items": [
    { "salePrice": 500, "category": "beauty", "quantity": 2 },
    { "salePrice": 300, "category": "fashion", "quantity": 1 }
  ],
  "overrides": {
    "commissionRate": 0.05
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `items` | array | Yes | รายการสินค้า |
| `items[].salePrice` | number | Yes | ราคาขายต่อชิ้น |
| `items[].category` | string | Yes | หมวดหมู่สินค้า |
| `items[].quantity` | integer | Yes | จำนวน |
| `overrides` | object | No | ปรับอัตราค่าธรรมเนียมแบบกำหนดเอง |

**Response (200)**

```json
{
  "items": [...],
  "result": {
    "commission": "65.00",
    "commissionRate": "5.00",
    "paymentFee": "42.72",
    "commerceGrowthFee": "0.00",
    "infrastructureFee": "0.00",
    "liveSpecialsFee": "0.00",
    "eamsFee": "0.00",
    "preorderFee": "0.00",
    "totalFees": "107.72",
    "effectiveRate": "9.79",
    "itemCount": 3
  }
}
```

---

## Fee Structure (TikTok Shop Thailand)

### Default Rates

| Fee Type | Rate | Description |
|----------|------|-------------|
| Commission | 7.49% | ค่าคอมมิชชั่น TikTok |
| Payment Fee | 3.21% | ค่าธรรมเนียมชำระเงิน |
| Pre-order | 2.00% | ค่าธรรมเนียมพรีออเดอร์ |

### Optional Fees (ผ่าน overrides)

| Fee Type | Override Field | Description |
|----------|---------------|-------------|
| Commerce Growth | `commerceGrowthRate` | ค่า Commerce Growth |
| Infrastructure | `infrastructureRate` | ค่าโครงสร้างพื้นฐาน |
| LIVE Specials | `enableLiveSpecials`, `liveSpecialsRate` | ค่า LIVE Specials |
| EAMS | `enableEams`, `eamsRate` | ค่า EAMS |
| Pre-order | `enablePreorder`, `preorderRate` | ค่าพรีออเดอร์ |

### Rate Types

ทุก fee override สนับสนุน 2 rate types:
- `PERCENTAGE` (default): คิดเป็น % ของราคาขาย
- `FIXED`: คิดเป็นจำนวนคงที่ (บาท) ต่อรายการ

---

## Error Responses

| Status | Error | Description |
|--------|-------|-------------|
| 400 | `invalid json` | JSON body ไม่ถูกต้อง |
| 400 | `sale price cannot be negative` | ราคาขายติดลบ |
| 400 | `items required` | batch request ไม่มี items |
| 401 | `unauthorized` | ไม่มี/ผิด token |
| 403 | `forbidden` | ไม่มีสิทธิ์ |

---

## Tenant Scope

- ใช้ `shop_id` จาก JWT auth context เสมอ
- ไม่รับ `shop_id` จาก body/query เป็น scope

---

## Acceptance Criteria

- [ ] POST /api/calculator/fees คำนวณค่าธรรมเนียมถูกต้องตาม default rates
- [ ] POST /api/calculator/fees รองรับ overrides สำหรับทุก fee type
- [ ] POST /api/calculator/fees/batch คำนวณค่าธรรมเนียมรวมสำหรับหลายรายการ
- [ ] ใช้ PERCENTAGE และ FIXED rate types ได้ถูกต้อง
- [ ] ทุก endpoint ผ่าน Auth + Tenant middleware
- [ ] Error responses ใช้ fixed messages จาก secure.go
