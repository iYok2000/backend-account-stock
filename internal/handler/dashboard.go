package handler

import (
	"net/http"
	"strconv"
	"time"

	"account-stock-be/internal/auth"
	"account-stock-be/internal/database"
	"account-stock-be/internal/middleware"

	"gorm.io/gorm"
)

const lowStockThreshold = 5

var bangkok = time.FixedZone("Asia/Bangkok", 7*3600)

type importDashboardRow struct {
	ShopID      string    `json:"shop_id"`
	SKUID       string    `json:"sku_id"`
	ProductName string    `json:"product_name"`
	Date        time.Time `json:"date"`
	Quantity    float64   `json:"quantity"`
	Revenue     float64   `json:"revenue"`
	Deductions  float64   `json:"deductions"`
	Refund      float64   `json:"refund"`
	Net         float64   `json:"net"`
}

type affiliateDashboardRow struct {
	SKUID            string    `json:"sku_id"`
	ProductName      string    `json:"product_name"`
	OrderDate        time.Time `json:"order_date"`
	CommissionAmount float64   `json:"commission_amount"`
	IneligibleAmount float64   `json:"ineligible_amount"`
	ItemsSold        float64   `json:"items_sold"`
}

func loadImports(db *gorm.DB, shopIDs []string) ([]importDashboardRow, error) {
	q := db.Table("import_sku_row").
		Select("shop_id, sku_id, product_name, date, quantity, revenue, deductions, refund, net").
		Where("deleted_at IS NULL")
	if len(shopIDs) > 0 {
		q = q.Where("shop_id IN ?", shopIDs)
	}
	var rows []importDashboardRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func loadImportsBetween(db *gorm.DB, shopIDs []string, start, end time.Time) ([]importDashboardRow, error) {
	q := db.Table("import_sku_row").
		Select("shop_id, sku_id, product_name, date, quantity, revenue, deductions, refund, net").
		Where("deleted_at IS NULL").
		Where("date BETWEEN ? AND ?", start, end)
	if len(shopIDs) > 0 {
		q = q.Where("shop_id IN ?", shopIDs)
	}
	var rows []importDashboardRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func loadAffiliateBetween(db *gorm.DB, companyID, userID string, start, end time.Time) ([]affiliateDashboardRow, error) {
	var rows []affiliateDashboardRow
	q := db.Table("affiliate_sku_row").
		Select("sku_id, product_name, order_date, commission_amount, ineligible_amount, items_sold").
		Where("company_id = ?", companyID)
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if !start.IsZero() || !end.IsZero() {
		// apply date filter only when start/end provided; keeps rowsที่ order_date เป็น NULL ไว้ใช้งาน summary
		if end.IsZero() {
			end = time.Now().In(bangkok)
		}
		if start.IsZero() {
			start = time.Time{}
		}
		q = q.Where("order_date BETWEEN ? AND ?", start, end)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// DashboardOverview handles GET /api/dashboard/overview
// Returns total products (distinct SKU) and low-stock count using latest quantity per SKU.
func DashboardOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ctx := middleware.GetContext(r.Context())
	if ctx == nil {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	db := database.DB()
	if db == nil {
		middleware.WriteJSONErrorMsg(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	// Affiliate ใช้ตาราง affiliate_sku_row scoped ด้วย company_id
	if ctx.Role == auth.RoleAffiliate {
		if ctx.CompanyID == "" {
			middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
			return
		}
		var totalProducts int64
		if err := db.Table("affiliate_sku_row").
			Where("company_id = ?", ctx.CompanyID).
			Distinct("sku_id").
			Count(&totalProducts).Error; err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		var lastImport *time.Time
		if err := db.Table("affiliate_sku_row").
			Where("company_id = ?", ctx.CompanyID).
			Select("MAX(order_date)").
			Scan(&lastImport).Error; err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"totalProducts": totalProducts,
			"lowStock":      0, // ไม่ใช้สำหรับ affiliate
			"lastImport":    formatDate(lastImport),
		})
		return
	}

	// Seller/Root: รวมทุก shop ที่ user เข้าถึง (shopId เดียวหรือทุก shop ใน company)
	shopIDs, err := shopIDsForContext(db, ctx)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	if len(shopIDs) == 0 {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	var totalProducts int64
	if err := db.Table("import_sku_row").
		Where("shop_id IN ?", shopIDs).
		Distinct("sku_id").
		Count(&totalProducts).Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	// load all rows for these shops
	rows, err := loadImports(db, shopIDs)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	latest := make(map[string]importDashboardRow)
	var lowStock int64
	var lastImport *time.Time
	for _, r := range rows {
		if lastImport == nil || r.Date.After(*lastImport) {
			t := r.Date
			lastImport = &t
		}
		if _, ok := latest[r.SKUID]; !ok {
			latest[r.SKUID] = r
			if r.Quantity <= lowStockThreshold {
				lowStock++
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"totalProducts": totalProducts,
		"lowStock":      lowStock,
		"lastImport":    formatDate(lastImport),
	})
}

// DashboardRevenue7d handles GET /api/dashboard/revenue-7d
// Returns last 7 days revenue and units (including days with zero).
func DashboardRevenue7d(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ctx := middleware.GetContext(r.Context())
	if ctx == nil {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	db := database.DB()
	if db == nil {
		middleware.WriteJSONErrorMsg(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	now := time.Now().In(bangkok)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, bangkok)
	start := today.AddDate(0, 0, -6)

	// Affiliate: ใช้ affiliate_sku_row scope ตาม company_id + user_id + order_date
	if ctx.Role == auth.RoleAffiliate {
		if ctx.CompanyID == "" || ctx.UserID == "" {
			middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
			return
		}
		affRows, err := loadAffiliateBetween(db, ctx.CompanyID, ctx.UserID, start, today)
		if err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		out := make([]map[string]interface{}, 0, 7)
		rowMap := make(map[string]struct {
			Revenue float64
			Units   float64
		})
		for _, r := range affRows {
			key := r.OrderDate.Format("2006-01-02")
			cur := rowMap[key]
			cur.Revenue += r.CommissionAmount
			cur.Units += r.ItemsSold
			rowMap[key] = cur
		}
		for i := 0; i < 7; i++ {
			d := start.AddDate(0, 0, i)
			key := d.Format("2006-01-02")
			if v, ok := rowMap[key]; ok {
				out = append(out, map[string]interface{}{
					"date":    key,
					"revenue": v.Revenue,
					"units":   v.Units,
				})
			} else {
				out = append(out, map[string]interface{}{
					"date":    key,
					"revenue": 0,
					"units":   0,
				})
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data": out,
			"from": start.Format("2006-01-02"),
			"to":   today.Format("2006-01-02"),
		})
		return
	}

	shopIDs, err := shopIDsForContext(db, ctx)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	if len(shopIDs) == 0 {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	rows, err := loadImportsBetween(db, shopIDs, start, today)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	// fill missing days
	out := make([]map[string]interface{}, 0, 7)
	rowMap := make(map[string]struct {
		Revenue float64
		Units   float64
	})
	for _, rrow := range rows {
		key := rrow.Date.Format("2006-01-02")
		cur := rowMap[key]
		cur.Revenue += rrow.Revenue
		cur.Units += rrow.Quantity
		rowMap[key] = cur
	}
	for i := 0; i < 7; i++ {
		d := start.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		if v, ok := rowMap[key]; ok {
			out = append(out, map[string]interface{}{
				"date":    key,
				"revenue": v.Revenue,
				"units":   v.Units,
			})
		} else {
			out = append(out, map[string]interface{}{
				"date":    key,
				"revenue": 0,
				"units":   0,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": out,
		"from": start.Format("2006-01-02"),
		"to":   today.Format("2006-01-02"),
	})
}

// DashboardLowStock handles GET /api/dashboard/low-stock?limit=10
// Returns latest quantity per SKU that is at/below threshold.
func DashboardLowStock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ctx := middleware.GetContext(r.Context())
	if ctx == nil {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	db := database.DB()
	if db == nil {
		middleware.WriteJSONErrorMsg(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	// ใช้ 5 เป็นค่าเริ่มต้นสำหรับ "top earning products" (affiliate) และ low stock (seller)
	limit := 5
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	if ctx.Role == auth.RoleAffiliate {
		if ctx.CompanyID == "" || ctx.UserID == "" {
			middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
			return
		}
		type row struct {
			SKU    string     `json:"sku"`
			Name   string     `json:"name"`
			Amount float64    `json:"amount"`
			Units  float64    `json:"units"`
			Date   *time.Time `json:"date"`
		}
		var out []row
		if err := db.Table("affiliate_sku_row").
			Select("sku_id as sku, COALESCE(NULLIF(product_name,''), sku_id) as name, SUM(commission_amount) as amount, SUM(items_sold) as units, MAX(order_date) as date").
			Where("company_id = ? AND user_id = ?", ctx.CompanyID, ctx.UserID).
			Group("sku_id, name").
			Order("amount DESC").
			Limit(limit).
			Scan(&out).Error; err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data":  out,
			"limit": limit,
		})
		return
	}

	shopIDs, err := shopIDsForContext(db, ctx)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	if len(shopIDs) == 0 {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	type row struct {
		SKU  string    `json:"sku"`
		Name string    `json:"name"`
		Qty  float64   `json:"qty"`
		Date time.Time `json:"date"`
	}
	allRows, err := loadImports(db, shopIDs)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	latest := make(map[string]importDashboardRow)
	for _, r := range allRows {
		if cur, ok := latest[r.SKUID]; !ok || r.Date.After(cur.Date) {
			latest[r.SKUID] = r
		}
	}
	var rows []row
	for _, v := range latest {
		if v.Quantity <= lowStockThreshold {
			rows = append(rows, row{SKU: v.SKUID, Name: firstNonEmpty(v.ProductName, v.SKUID), Qty: v.Quantity, Date: v.Date})
		}
	}
	// sort ascending by qty then sku
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].Qty < rows[i].Qty || (rows[j].Qty == rows[i].Qty && rows[j].SKU < rows[i].SKU) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  rows,
		"limit": limit,
	})
}

// DashboardKPIs handles GET /api/dashboard/kpis
// Returns revenue/discount/net aggregates; order count unavailable (null).
func DashboardKPIs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ctx := middleware.GetContext(r.Context())
	if ctx == nil {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	db := database.DB()
	if db == nil {
		middleware.WriteJSONErrorMsg(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	// Affiliate KPI ใช้ commission_amount เป็น revenue, ineligible_amount เป็น discount
	if ctx.Role == auth.RoleAffiliate {
		if ctx.CompanyID == "" || ctx.UserID == "" {
			middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
			return
		}
		// ไม่กรองวันที่ เพื่อให้รวมทุกออร์เดอร์ที่มี (บางไฟล์ไม่มี order_date)
		rows, err := loadAffiliateBetween(db, ctx.CompanyID, ctx.UserID, time.Time{}, time.Time{})
		if err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		var revenue, inelig float64
		for _, r := range rows {
			revenue += r.CommissionAmount
			inelig += r.IneligibleAmount
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"totalRevenue":  revenue,
			"totalDiscount": inelig,
			"netBase":       revenue - inelig,
			"totalOrders":   nil,
		})
		return
	}

	shopIDs, err := shopIDsForContext(db, ctx)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	if len(shopIDs) == 0 {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	rows, err := loadImports(db, shopIDs)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	var revenue, discount, net float64
	for _, r := range rows {
		revenue += r.Revenue
		discount += r.Deductions
		net += r.Net
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"totalRevenue":  revenue,
		"totalDiscount": discount,
		"netBase":       net,
		"totalOrders":   nil, // no order feed yet
	})
}
