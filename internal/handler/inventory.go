package handler

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"account-stock-be/internal/database"
	"account-stock-be/internal/middleware"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxImportItems   = 50000
	maxNumericAmount = 1e12 // guardrail for revenue/deductions/net
)

var errInvalidAmount = errors.New("invalid amount")

// ImportInventoryRequest matches FE payload (items always present; summary/daily optional).
type ImportInventoryRequest struct {
	Tier    string             `json:"tier"`
	Items   []ImportSKURequest `json:"items"`
	Summary interface{}        `json:"summary,omitempty"`
	Daily   interface{}        `json:"daily,omitempty"`
}

type ImportSKURequest struct {
	Date        string   `json:"date"`
	SKUID       string   `json:"sku_id"`
	SellerSKU   string   `json:"seller_sku"`
	ProductName string   `json:"product_name"`
	Variation   string   `json:"variation"`
	Quantity    float64  `json:"quantity"`
	Revenue     float64  `json:"revenue"`
	Deductions  float64  `json:"deductions"`
	Refund      float64  `json:"refund"`
	Net         float64  `json:"net"`
	Name        string   `json:"name"` // alias from FE (fallback for product name)
	_           struct{} `json:"-"`    // no extra fields
}

type importSKURow struct {
	ID          string    `gorm:"column:id;type:uuid;default:gen_random_uuid()" json:"id"`
	ShopID      string    `gorm:"column:shop_id" json:"-"`
	Date        time.Time `gorm:"column:date" json:"date"`
	SKUID       string    `gorm:"column:sku_id" json:"sku_id"`
	SellerSKU   string    `gorm:"column:seller_sku" json:"seller_sku"`
	ProductName string    `gorm:"column:product_name" json:"product_name"`
	Variation   string    `gorm:"column:variation" json:"variation"`
	Quantity    float64   `gorm:"column:quantity" json:"quantity"`
	Revenue     float64   `gorm:"column:revenue" json:"revenue"`
	Deductions  float64   `gorm:"column:deductions" json:"deductions"`
	Refund      float64   `gorm:"column:refund" json:"refund"`
	Net         float64   `gorm:"column:net" json:"net"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (importSKURow) TableName() string { return "import_sku_row" }

// ImportInventory handles POST /api/inventory/import
// Auth + inventory:create + tenant scope (shop_id from auth).
func ImportInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ctx := middleware.GetContext(r.Context())
	if ctx == nil || ctx.ShopID == "" {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	var body ImportInventoryRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
		return
	}
	if body.Tier != "free" && body.Tier != "paid" {
		middleware.WriteJSONErrorMsg(w, "tier must be free or paid", http.StatusBadRequest)
		return
	}
	if len(body.Items) == 0 {
		middleware.WriteJSONErrorMsg(w, "items is required", http.StatusBadRequest)
		return
	}
	if len(body.Items) > maxImportItems {
		middleware.WriteJSONErrorMsg(w, "items too many (max 50000)", http.StatusBadRequest)
		return
	}

	db := database.DB()
	if db == nil {
		middleware.WriteJSONErrorMsg(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()
	rows := make([]importSKURow, 0, len(body.Items))
	for _, it := range body.Items {
		row, err := validateImportItem(it)
		if err != nil {
			middleware.WriteJSONErrorMsg(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.ShopID = ctx.ShopID
		row.CreatedAt = now
		row.UpdatedAt = now
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		middleware.WriteJSONErrorMsg(w, "no valid items", http.StatusBadRequest)
		return
	}

	// Upsert by (shop_id, date, sku_id)
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "shop_id"}, {Name: "date"}, {Name: "sku_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"seller_sku":   gorm.Expr("EXCLUDED.seller_sku"),
			"product_name": gorm.Expr("EXCLUDED.product_name"),
			"variation":    gorm.Expr("EXCLUDED.variation"),
			"quantity":     gorm.Expr("EXCLUDED.quantity"),
			"revenue":      gorm.Expr("EXCLUDED.revenue"),
			"deductions":   gorm.Expr("EXCLUDED.deductions"),
			"refund":       gorm.Expr("EXCLUDED.refund"),
			"net":          gorm.Expr("EXCLUDED.net"),
			"updated_at":   gorm.Expr("now()"),
		}),
	}).Create(&rows)
	if tx.Error != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":       true,
		"imported": tx.RowsAffected, // includes upserts
		"updated":  nil,             // not tracked separately
	})
}

// InventoryList handles GET /api/inventory
// Returns SKU rows for current shop.
func InventoryList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ctx := middleware.GetContext(r.Context())
	if ctx == nil || ctx.ShopID == "" {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	db := database.DB()
	if db == nil {
		middleware.WriteJSONErrorMsg(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	limit, offset := clampLimitAndOffset(r.URL.Query())

	var rows []importSKURow
	if err := db.Where("shop_id = ?", ctx.ShopID).
		Order("date DESC, sku_id").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	type item struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		SKU        string  `json:"sku"`
		Quantity   float64 `json:"quantity"`
		Status     string  `json:"status"`
		Date       string  `json:"date"`
		Revenue    float64 `json:"revenue"`
		Deductions float64 `json:"deductions"`
		Refund     float64 `json:"refund"`
		Net        float64 `json:"net"`
	}
	out := make([]item, 0, len(rows))
	for _, rrow := range rows {
		status := statusFromQty(rrow.Quantity)
		out = append(out, item{
			ID:         rrow.ID,
			Name:       firstNonEmpty(rrow.ProductName, rrow.SKUID),
			SKU:        rrow.SKUID,
			Quantity:   rrow.Quantity,
			Status:     status,
			Date:       rrow.Date.Format("2006-01-02"),
			Revenue:    rrow.Revenue,
			Deductions: rrow.Deductions,
			Refund:     rrow.Refund,
			Net:        rrow.Net,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":   out,
		"total":  len(out),
		"limit":  limit,
		"offset": offset,
	})
}

// InventorySummary handles GET /api/inventory/summary?period=current_month
func InventorySummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ctx := middleware.GetContext(r.Context())
	if ctx == nil || ctx.ShopID == "" {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	db := database.DB()
	if db == nil {
		middleware.WriteJSONErrorMsg(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "current_month"
	}
	startDate := periodStart(period)
	var lastImport *time.Time
	type agg struct {
		UniqueSkus int64
		Units      float64
		Revenue    float64
		Net        float64
		LastDate   *time.Time
	}
	var a agg
	if err := db.Table("import_sku_row").
		Select("COUNT(DISTINCT sku_id) AS unique_skus, COALESCE(SUM(quantity),0) AS units, COALESCE(SUM(revenue),0) AS revenue, COALESCE(SUM(net),0) AS net, MAX(date) AS last_date").
		Where("shop_id = ? AND date >= ?", ctx.ShopID, startDate).
		Scan(&a).Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	lastImport = a.LastDate

	// top 5 skus by revenue
	type top struct {
		SKU      string  `json:"sku"`
		Name     string  `json:"name"`
		Quantity float64 `json:"quantity"`
		Revenue  float64 `json:"revenue"`
		Net      float64 `json:"net"`
		Date     string  `json:"date"`
	}
	var topRows []top
	if err := db.Table("import_sku_row").
		Select("sku_id AS sku, COALESCE(MAX(product_name), sku_id) AS name, SUM(quantity) AS quantity, SUM(revenue) AS revenue, SUM(net) AS net, MAX(date)::text AS date").
		Where("shop_id = ? AND date >= ?", ctx.ShopID, startDate).
		Group("sku_id").
		Order("revenue DESC NULLS LAST").
		Limit(5).
		Scan(&topRows).Error; err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"uniqueSkus":       a.UniqueSkus,
		"unitsThisMonth":   a.Units,
		"revenueThisMonth": a.Revenue,
		"netThisMonth":     a.Net,
		"lastImport":       formatDate(lastImport),
		"topSkus":          topRows,
	})
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("date is required")
	}
	// Accept YYYY-MM-DD
	return time.Parse("2006-01-02", s)
}

func validateImportItem(it ImportSKURequest) (importSKURow, error) {
	sku := strings.TrimSpace(it.SKUID)
	if sku == "" {
		return importSKURow{}, errors.New("sku_id is required")
	}
	d, err := parseDate(it.Date)
	if err != nil {
		return importSKURow{}, err
	}
	if !validAmount(it.Quantity, true) {
		return importSKURow{}, errors.New("quantity must be >= 0 and reasonable")
	}
	if !validAmount(it.Revenue, true) || !validAmount(it.Deductions, true) || !validAmount(it.Refund, true) {
		return importSKURow{}, errors.New("revenue, deductions, and refund must be >= 0 and reasonable")
	}
	if math.Abs(it.Net) > maxNumericAmount {
		return importSKURow{}, errors.New("net amount out of range")
	}

	name := strings.TrimSpace(it.ProductName)
	if name == "" {
		name = strings.TrimSpace(it.Name)
	}
	return importSKURow{
		Date:        d,
		SKUID:       sku,
		SellerSKU:   strings.TrimSpace(it.SellerSKU),
		ProductName: name,
		Variation:   strings.TrimSpace(it.Variation),
		Quantity:    it.Quantity,
		Revenue:     it.Revenue,
		Deductions:  it.Deductions,
		Refund:      it.Refund,
		Net:         it.Net,
	}, nil
}

func statusFromQty(q float64) string {
	if q <= 0 {
		return "out_of_stock"
	}
	if q <= 5 {
		return "low_stock"
	}
	return "in_stock"
}

func validAmount(v float64, allowZero bool) bool {
	if v < 0 {
		return false
	}
	if !allowZero && v == 0 {
		return false
	}
	return v <= maxNumericAmount
}

func periodStart(period string) time.Time {
	bkk := time.FixedZone("Asia/Bangkok", 7*3600)
	now := time.Now().In(bkk)
	switch period {
	case "current_month":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, bkk)
	default:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, bkk)
	}
}

func formatDate(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func clampLimitAndOffset(q url.Values) (int, int) {
	const defaultLimit = 100
	const maxLimit = 5000
	limit := defaultLimit
	offset := 0
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return limit, offset
}
