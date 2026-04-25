package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"account-stock-be/internal/database"
	"account-stock-be/internal/middleware"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AffiliateImportRequest payload: items from FE after processing affiliate file.
type AffiliateImportRequest struct {
	Items []AffiliateItem `json:"items"`
}

// AffiliateItem matches affiliate_sku_row columns used by dashboard/analytics.
type AffiliateItem struct {
	AffiliateShop      string  `json:"affiliate_shop"`
	OrderID            string  `json:"order_id"`
	SettlementStatus   string  `json:"settlement_status"`
	SKUID              string  `json:"sku_id"`
	ProductName        string  `json:"product_name"`
	ItemsSold          float64 `json:"items_sold"`
	GMV                float64 `json:"gmv"`
	CommissionAmount   float64 `json:"commission_amount"`   // Total final earned amount (realized)
	StandardCommission float64 `json:"standard_commission"` // Est. standard commission
	ShopAdsCommission  float64 `json:"shop_ads_commission"` // Est. Shop Ads commission
	CommissionBase     float64 `json:"commission_base"`
	CommissionRate     float64 `json:"commission_rate"`
	IneligibleAmount   float64 `json:"ineligible_amount"` // missing commissions
	ContentType        string  `json:"content_type"`
	OrderDate          string  `json:"order_date"`      // YYYY-MM-DD
	SettlementDate     string  `json:"settlement_date"` // YYYY-MM-DD
}

type affiliateRow struct {
	ID                 string     `gorm:"column:id;type:uuid;default:gen_random_uuid()"`
	CompanyID          string     `gorm:"column:company_id"`
	UserID             string     `gorm:"column:user_id"`
	Date               time.Time  `gorm:"column:date"`
	AffiliateShop      string     `gorm:"column:affiliate_shop"`
	OrderID            string     `gorm:"column:order_id"`
	SettlementStatus   string     `gorm:"column:settlement_status"`
	SKUID              string     `gorm:"column:sku_id"`
	ProductName        string     `gorm:"column:product_name"`
	ItemsSold          float64    `gorm:"column:items_sold"`
	GMV                float64    `gorm:"column:gmv"`
	CommissionAmount   float64    `gorm:"column:commission_amount"`
	StandardCommission float64    `gorm:"column:standard_commission"`
	ShopAdsCommission  float64    `gorm:"column:shop_ads_commission"`
	CommissionBase     float64    `gorm:"column:commission_base"`
	CommissionRate     float64    `gorm:"column:commission_rate"`
	IneligibleAmount   float64    `gorm:"column:ineligible_amount"`
	ContentType        string     `gorm:"column:content_type"`
	OrderDate          *time.Time `gorm:"column:order_date"`
	SettlementDate     *time.Time `gorm:"column:settlement_date"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

func (affiliateRow) TableName() string { return "affiliate_sku_row" }

const maxAffiliateItems = 50000

// parseDateFlexible accepts common date formats and returns (time, ok)
func parseDateFlexible(s string) (time.Time, bool) {
	layouts := []string{
		"2006-01-02", "2006/01/02",
		"2006-1-2", "2006/1/2",
		"02/01/2006", "02-01-2006",
		"2/1/2006", "2-1-2006",
		"02/01/06", "02-01-06", // fallback two-digit year
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// AffiliateImport handles POST /api/affiliate/import
// Auth + inventory:create (Affiliate role has this). Scoped by company_id from JWT.
func AffiliateImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ctx := middleware.GetContext(r.Context())
	if ctx == nil || ctx.CompanyID == "" || ctx.UserID == "" {
		middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	var body AffiliateImportRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
		return
	}
	if len(body.Items) == 0 {
		middleware.WriteJSONErrorMsg(w, "items is required", http.StatusBadRequest)
		return
	}
	if len(body.Items) > maxAffiliateItems {
		middleware.WriteJSONErrorMsg(w, "items too many (max 50000)", http.StatusBadRequest)
		return
	}
	db := database.DB()
	if db == nil {
		middleware.WriteJSONErrorMsg(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()
	rows := make([]affiliateRow, 0, len(body.Items))
	for _, it := range body.Items {
		if it.OrderID == "" || it.SKUID == "" {
			middleware.WriteJSONErrorMsg(w, "order_id and sku_id are required", http.StatusBadRequest)
			return
		}
		orderDateStr := it.OrderDate
		if orderDateStr == "" {
			t := time.Now().In(tzBangkok)
			orderDateStr = t.Format("2006-01-02")
		}
		settleDateStr := it.SettlementDate

		ar := affiliateRow{
			CompanyID:          ctx.CompanyID,
			UserID:             ctx.UserID,
			AffiliateShop:      it.AffiliateShop,
			OrderID:            it.OrderID,
			SettlementStatus:   it.SettlementStatus,
			SKUID:              it.SKUID,
			ProductName:        it.ProductName,
			ItemsSold:          it.ItemsSold,
			GMV:                it.GMV,
			CommissionAmount:   it.CommissionAmount,
			StandardCommission: it.StandardCommission,
			ShopAdsCommission:  it.ShopAdsCommission,
			CommissionBase:     it.CommissionBase,
			CommissionRate:     it.CommissionRate,
			IneligibleAmount:   it.IneligibleAmount,
			ContentType:        it.ContentType,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if orderDateStr != "" {
			if t, ok := parseDateFlexible(orderDateStr); ok {
				ar.OrderDate = &t
				ar.Date = t
			}
		}
		if settleDateStr != "" {
			if t, ok := parseDateFlexible(settleDateStr); ok {
				ar.SettlementDate = &t
			}
		}
		if ar.OrderDate == nil {
			t := time.Now().In(tzBangkok)
			d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, tzBangkok)
			ar.OrderDate = &d
			ar.Date = d
		}
		if ar.Date.IsZero() && ar.OrderDate != nil {
			ar.Date = *ar.OrderDate
		}
		// Double-safety: Date must never be null (DB NOT NULL)
		if ar.Date.IsZero() {
			t := time.Now().In(tzBangkok)
			ar.Date = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, tzBangkok)
			if ar.OrderDate == nil {
				od := ar.Date
				ar.OrderDate = &od
			}
		}
		rows = append(rows, ar)
	}

	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "company_id"},
			{Name: "user_id"},
			{Name: "order_id"},
			{Name: "sku_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"affiliate_shop":      gorm.Expr("EXCLUDED.affiliate_shop"),
			"settlement_status":   gorm.Expr("EXCLUDED.settlement_status"),
			"product_name":        gorm.Expr("EXCLUDED.product_name"),
			"items_sold":          gorm.Expr("EXCLUDED.items_sold"),
			"gmv":                 gorm.Expr("EXCLUDED.gmv"),
			"commission_amount":   gorm.Expr("EXCLUDED.commission_amount"),
			"standard_commission": gorm.Expr("EXCLUDED.standard_commission"),
			"shop_ads_commission": gorm.Expr("EXCLUDED.shop_ads_commission"),
			"commission_base":     gorm.Expr("EXCLUDED.commission_base"),
			"commission_rate":     gorm.Expr("EXCLUDED.commission_rate"),
			"ineligible_amount":   gorm.Expr("EXCLUDED.ineligible_amount"),
			"content_type":        gorm.Expr("EXCLUDED.content_type"),
			"order_date":          gorm.Expr("EXCLUDED.order_date"),
			"settlement_date":     gorm.Expr("EXCLUDED.settlement_date"),
			"updated_at":          gorm.Expr("now()"),
		}),
	}).Create(&rows)
	if tx.Error != nil {
		// Friendly errors for common data issues
		errMsg := tx.Error.Error()
		if strings.Contains(errMsg, "null value in column \"date\"") {
			middleware.WriteJSONErrorMsg(w, "ข้อมูลวันที่ไม่ครบหรือรูปแบบไม่ถูกต้อง (ต้องเป็น YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":       true,
		"imported": tx.RowsAffected,
	})
}
