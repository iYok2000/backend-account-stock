package handler

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"account-stock-be/internal/auth"
	"account-stock-be/internal/database"
	"account-stock-be/internal/middleware"
	"gorm.io/gorm"
)

var tzBangkok = time.FixedZone("Asia/Bangkok", 7*3600)

// ─── DB row types ─────────────────────────────────────────────────────────────

type importRow struct {
	Date        time.Time `gorm:"column:date"`
	Revenue     float64   `gorm:"column:revenue"`
	Net         float64   `gorm:"column:net"`
	Deductions  float64   `gorm:"column:deductions"`
	Refund      float64   `gorm:"column:refund"`
	Quantity    float64   `gorm:"column:quantity"`
	SKUID       string    `gorm:"column:sku_id"`
	ProductName string    `gorm:"column:product_name"`
	ShopID      string    `gorm:"column:shop_id"`
}

type affiliateAnalyticsRow struct {
	OrderDate        time.Time `gorm:"column:order_date"`
	GMV              float64   `gorm:"column:gmv"`
	CommissionAmount float64   `gorm:"column:commission_amount"`
	IneligibleAmount float64   `gorm:"column:ineligible_amount"`
	ItemsSold        float64   `gorm:"column:items_sold"`
	SKUID            string    `gorm:"column:sku_id"`
	ProductName      string    `gorm:"column:product_name"`
}

// ─── Aggregation structs ──────────────────────────────────────────────────────

type importAgg struct {
	GMV        float64
	Settlement float64
	Deductions float64
	Refund     float64
}

type affiliateAgg struct {
	GMV        float64
	Earned     float64
	Ineligible float64
}

type dailyMetric struct {
	revenue    float64
	profit     float64
	settlement float64
}

type skuMetric struct {
	name     string
	quantity float64
	revenue  float64
	profit   float64
}

// ─── Fetch helpers ────────────────────────────────────────────────────────────

func shopIDsForContext(db *gorm.DB, ctx *auth.Context) ([]string, error) {
	if ctx == nil {
		return nil, nil
	}
	if ctx.ShopID != "" {
		return []string{ctx.ShopID}, nil
	}
	if ctx.CompanyID != "" {
		var ids []string
		if err := db.Table("shops").Select("id").Where("company_id = ?", ctx.CompanyID).Pluck("id", &ids).Error; err != nil {
			return nil, err
		}
		return ids, nil
	}
	return nil, nil
}

func fetchImportRows(db *gorm.DB, shopIDs []string, from, to time.Time) ([]importRow, error) {
	query := db.Table("import_sku_row").
		Select("date, revenue, net, deductions, refund, quantity, sku_id, product_name, shop_id").
		Where("deleted_at IS NULL").
		Where("date BETWEEN ? AND ?", from, to)
	if len(shopIDs) > 0 {
		query = query.Where("shop_id IN ?", shopIDs)
	}
	var rows []importRow
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func fetchAffiliateRows(db *gorm.DB, companyID, userID string, from, to time.Time) ([]affiliateAnalyticsRow, error) {
	var rows []affiliateAnalyticsRow
	if err := db.Table("affiliate_sku_row").
		Select("order_date, gmv, commission_amount, ineligible_amount, items_sold, sku_id, product_name").
		Where("company_id = ? AND user_id = ? AND order_date BETWEEN ? AND ?", companyID, userID, from, to).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func parseDateRange(r *http.Request) (time.Time, time.Time) {
	end := time.Now().In(tzBangkok).Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -29)
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			start = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, tzBangkok)
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, tzBangkok)
		}
	}
	if end.Before(start) {
		start, end = end, start
	}
	return start, end
}

func parsePeriod(r *http.Request) string {
	switch r.URL.Query().Get("period") {
	case "weekly":
		return "weekly"
	case "monthly":
		return "monthly"
	}
	return "monthly"
}

// ─── Go-side aggregation ──────────────────────────────────────────────────────

func aggregateImport(rows []importRow) importAgg {
	var a importAgg
	for _, r := range rows {
		a.GMV += r.Revenue
		a.Settlement += r.Net
		a.Deductions += r.Deductions
		a.Refund += r.Refund
	}
	return a
}

func aggregateAffiliate(rows []affiliateAnalyticsRow) affiliateAgg {
	var a affiliateAgg
	for _, r := range rows {
		a.GMV += r.GMV
		a.Earned += r.CommissionAmount
		a.Ineligible += r.IneligibleAmount
	}
	return a
}

func importToDailyMap(rows []importRow) map[string]*dailyMetric {
	m := make(map[string]*dailyMetric)
	for _, r := range rows {
		key := r.Date.Format("2006-01-02")
		d := m[key]
		if d == nil {
			d = &dailyMetric{}
			m[key] = d
		}
		d.revenue += r.Revenue
		d.profit += r.Net
		d.settlement += r.Net
	}
	return m
}

func affiliateToDailyMap(rows []affiliateAnalyticsRow) map[string]*dailyMetric {
	m := make(map[string]*dailyMetric)
	for _, r := range rows {
		key := r.OrderDate.Format("2006-01-02")
		d := m[key]
		if d == nil {
			d = &dailyMetric{}
			m[key] = d
		}
		d.revenue += r.CommissionAmount
		d.profit += r.CommissionAmount - r.IneligibleAmount
		d.settlement += r.CommissionAmount
	}
	return m
}

func importToSkuMap(rows []importRow) map[string]*skuMetric {
	m := make(map[string]*skuMetric)
	for _, r := range rows {
		s := m[r.SKUID]
		if s == nil {
			name := r.ProductName
			if name == "" {
				name = r.SKUID
			}
			s = &skuMetric{name: name}
			m[r.SKUID] = s
		}
		if r.ProductName != "" {
			s.name = r.ProductName
		}
		s.quantity += r.Quantity
		s.revenue += r.Revenue
		s.profit += r.Net
	}
	return m
}

func affiliateToSkuMap(rows []affiliateAnalyticsRow) map[string]*skuMetric {
	m := make(map[string]*skuMetric)
	for _, r := range rows {
		s := m[r.SKUID]
		if s == nil {
			name := r.ProductName
			if name == "" {
				name = r.SKUID
			}
			s = &skuMetric{name: name}
			m[r.SKUID] = s
		}
		if r.ProductName != "" {
			s.name = r.ProductName
		}
		s.quantity += r.ItemsSold
		s.revenue += r.CommissionAmount
		s.profit += r.CommissionAmount - r.IneligibleAmount
	}
	return m
}

// ─── Response writers ─────────────────────────────────────────────────────────

func writeDailyMetrics(w http.ResponseWriter, dayMap map[string]*dailyMetric, from, to time.Time) {
	days := int(to.Sub(from).Hours()/24) + 1
	timeSeries := make([]map[string]interface{}, 0, days)
	var totalRevenue, totalProfit float64
	for i := 0; i < days; i++ {
		d := from.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		if v, ok := dayMap[key]; ok {
			timeSeries = append(timeSeries, map[string]interface{}{
				"label":      key,
				"revenue":    v.revenue,
				"profit":     v.profit,
				"settlement": v.settlement,
			})
			totalRevenue += v.revenue
			totalProfit += v.profit
		} else {
			timeSeries = append(timeSeries, map[string]interface{}{
				"label":      key,
				"revenue":    0,
				"profit":     0,
				"settlement": 0,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"hasData":    totalRevenue > 0 || totalProfit > 0,
		"totals":     map[string]float64{"revenue": totalRevenue, "profit": totalProfit, "settlement": totalProfit},
		"timeSeries": timeSeries,
		"from":       from.Format("2006-01-02"),
		"to":         to.Format("2006-01-02"),
	})
}

func writeProductMetrics(w http.ResponseWriter, skuMap map[string]*skuMetric, from, to time.Time) {
	out := make([]map[string]interface{}, 0, len(skuMap))
	for skuID, s := range skuMap {
		var margin *float64
		if s.revenue > 0 {
			m := (s.profit / s.revenue) * 100
			margin = &m
		}
		out = append(out, map[string]interface{}{
			"skuId":        skuID,
			"name":         s.name,
			"category":     "general",
			"quantity":     s.quantity,
			"revenue":      s.revenue,
			"profit":       s.profit,
			"profitMargin": margin,
			"hasCost":      false,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"products": out,
		"hasData":  len(out) > 0,
		"from":     from.Format("2006-01-02"),
		"to":       to.Format("2006-01-02"),
	})
}

// ─── Trends: bucket by week/month ────────────────────────────────────────────

type trendBucket struct {
	key      string
	label    string
	revenue  float64
	orders   float64
	profit   float64
	discount float64
	days     int
}

func rowsToTrendBuckets(rows []importRow, period string) []trendBucket {
	if len(rows) == 0 {
		return nil
	}
	type acc struct {
		revenue  float64
		profit   float64
		discount float64
		days     map[string]struct{}
	}
	m := make(map[string]*acc)
	for _, r := range rows {
		var key string
		d := r.Date.In(tzBangkok)
		if period == "weekly" {
			year, week := d.ISOWeek()
			key = formatWeekKey(year, week)
		} else {
			key = d.Format("2006-01")
		}
		a := m[key]
		if a == nil {
			a = &acc{days: make(map[string]struct{})}
			m[key] = a
		}
		a.revenue += r.Revenue
		a.profit += r.Net
		a.discount += r.Deductions
		a.days[r.Date.Format("2006-01-02")] = struct{}{}
	}
	var out []trendBucket
	for k, a := range m {
		out = append(out, trendBucket{
			key:      k,
			label:    k,
			revenue:  a.revenue,
			orders:   0,
			profit:   a.profit,
			discount: a.discount,
			days:     len(a.days),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out
}

func formatWeekKey(year, week int) string {
	return fmt.Sprintf("%d-W%02d", year, week)
}

func rowsToMonthlyBuckets(rows []importRow) []trendBucket {
	return rowsToTrendBuckets(rows, "monthly")
}

func trendBucketsToMomGrowth(buckets []trendBucket) []map[string]interface{} {
	if len(buckets) < 2 {
		return nil
	}
	var out []map[string]interface{}
	for i := 1; i < len(buckets); i++ {
		prev := buckets[i-1].revenue
		curr := buckets[i].revenue
		growth := 0.0
		if prev > 0 {
			growth = ((curr - prev) / prev) * 100
		}
		out = append(out, map[string]interface{}{
			"label":  buckets[i].label,
			"growth": growth,
		})
	}
	return out
}

func trendBucketsToYoy(rows []importRow) []map[string]interface{} {
	// Group by month (ignore year), then for each month we need current year + previous year
	byMonth := make(map[string]map[int]float64) // "01" -> year -> revenue
	for _, r := range rows {
		d := r.Date.In(tzBangkok)
		monthKey := d.Format("01")
		year := d.Year()
		if byMonth[monthKey] == nil {
			byMonth[monthKey] = make(map[int]float64)
		}
		byMonth[monthKey][year] += r.Revenue
	}
	var out []map[string]interface{}
	now := time.Now().In(tzBangkok)
	curYear := now.Year()
	prevYear := curYear - 1
	for _, monthKey := range []string{"01", "02", "03", "04", "05", "06", "07", "08", "09", "10", "11", "12"} {
		years := byMonth[monthKey]
		if years[curYear] == 0 && years[prevYear] == 0 {
			continue
		}
		label := fmt.Sprintf("%s-%d", monthKey, curYear)
		out = append(out, map[string]interface{}{
			"label":        label,
			"currentYear":  years[curYear],
			"previousYear": years[prevYear],
		})
	}
	return out
}

func writeTrendsResponse(w http.ResponseWriter, buckets, monthlyBuckets []trendBucket, momGrowth, yoy []map[string]interface{}, from, to time.Time, period string) {
	bucketList := make([]map[string]interface{}, 0, len(buckets))
	for _, b := range buckets {
		bucketList = append(bucketList, map[string]interface{}{
			"key":      b.key,
			"label":    b.label,
			"revenue":  b.revenue,
			"orders":   b.orders,
			"profit":   b.profit,
			"discount": b.discount,
			"days":     b.days,
		})
	}
	monthList := make([]map[string]interface{}, 0, len(monthlyBuckets))
	for _, b := range monthlyBuckets {
		monthList = append(monthList, map[string]interface{}{
			"key":      b.key,
			"label":    b.label,
			"revenue":  b.revenue,
			"orders":   b.orders,
			"profit":   b.profit,
			"discount": b.discount,
			"days":     b.days,
		})
	}
	hasData := len(bucketList) > 0 || len(monthList) > 0 || len(momGrowth) > 0 || len(yoy) > 0
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"buckets":         bucketList,
		"monthlyBuckets":  monthList,
		"momGrowth":       momGrowth,
		"yoy":             yoy,
		"hasData":         hasData,
		"from":            from.Format("2006-01-02"),
		"to":              to.Format("2006-01-02"),
		"period":          period,
	})
}

// ─── Profitability: margin distribution from SKU map ───────────────────────────

func skuMapToProfitability(skuMap map[string]*skuMetric, rows []importRow, from, to time.Time) (avgMargin float64, marginBuckets []map[string]interface{}, byCategory []map[string]interface{}, marginTrend []map[string]interface{}) {
	if len(skuMap) == 0 {
		return 0, nil, nil, nil
	}
	var totalMargin float64
	var count int
	for _, s := range skuMap {
		if s.revenue > 0 {
			m := (s.profit / s.revenue) * 100
			totalMargin += m
			count++
		}
	}
	if count > 0 {
		avgMargin = totalMargin / float64(count)
	}
	// Margin buckets: <0, 0-15, 15-25, 25+
	ranges := []struct {
		label string
		min   float64
		max   float64
	}{
		{"<0%", -1e9, 0},
		{"0-15%", 0, 15},
		{"15-25%", 15, 25},
		{"25%+", 25, 1e9},
	}
	rangeCounts := make([]int, len(ranges))
	for _, s := range skuMap {
		if s.revenue <= 0 {
			continue
		}
		m := (s.profit / s.revenue) * 100
		for i, r := range ranges {
			if m >= r.min && m < r.max {
				rangeCounts[i]++
				break
			}
		}
	}
	for i, r := range ranges {
		marginBuckets = append(marginBuckets, map[string]interface{}{"range": r.label, "count": rangeCounts[i]})
	}
	// byCategory: we only have "general", so one row with total profit + avg margin
	var totalProfit float64
	for _, s := range skuMap {
		totalProfit += s.profit
	}
	catMargin := 0.0
	if len(skuMap) > 0 {
		catMargin = avgMargin
	}
	byCategory = []map[string]interface{}{
		{"category": "general", "profit": totalProfit, "margin": catMargin},
	}
	// marginTrend: aggregate by month from rows
	monthlyMargin := make(map[string][]float64)
	for _, r := range rows {
		if r.Revenue <= 0 {
			continue
		}
		m := (r.Net / r.Revenue) * 100
		key := r.Date.In(tzBangkok).Format("2006-01")
		monthlyMargin[key] = append(monthlyMargin[key], m)
	}
	for _, k := range sortedMapKeys(monthlyMargin) {
		vals := monthlyMargin[k]
		var sum float64
		for _, v := range vals {
			sum += v
		}
		avg := sum / float64(len(vals))
		marginTrend = append(marginTrend, map[string]interface{}{"label": k, "margin": avg})
	}
	sort.Slice(marginTrend, func(i, j int) bool {
		return marginTrend[i]["label"].(string) < marginTrend[j]["label"].(string)
	})
	return avgMargin, marginBuckets, byCategory, marginTrend
}

func sortedMapKeys(m map[string][]float64) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ─── GET /api/analytics/reconciliation ───────────────────────────────────────

func AnalyticsReconciliation(w http.ResponseWriter, r *http.Request) {
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
	from, to := parseDateRange(r)

	// Affiliate branch
	if ctx.Role == auth.RoleAffiliate {
		if ctx.CompanyID == "" || ctx.UserID == "" {
			middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
			return
		}
		rows, err := fetchAffiliateRows(db, ctx.CompanyID, ctx.UserID, from, to)
		if err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		a := aggregateAffiliate(rows)
		totalFees := a.Ineligible
		net := a.Earned - totalFees
		settlementRate := 0.0
		if a.GMV > 0 {
			settlementRate = (a.Earned / a.GMV) * 100
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"gmv":            a.GMV,
			"settlement":     a.Earned,
			"totalFees":      totalFees,
			"netProfit":      net,
			"settlementRate": settlementRate,
			"feeBreakdown":   []map[string]interface{}{{"label": "ineligible", "value": totalFees}},
			"from":           from.Format("2006-01-02"),
			"to":             to.Format("2006-01-02"),
		})
		return
	}

	// Owner / Admin / Root branch — shopIDsForContext handles all scoping
	shopIDs, err := shopIDsForContext(db, ctx)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	rows, err := fetchImportRows(db, shopIDs, from, to)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	a := aggregateImport(rows)
	totalFees := a.GMV - a.Settlement
	if totalFees < 0 {
		totalFees = 0
	}
	settlementRate := 0.0
	if a.GMV > 0 {
		settlementRate = (a.Settlement / a.GMV) * 100
	}
	// tiktokCommission = implied platform cut (GMV - settlement - deductions - refund)
	tiktokFee := totalFees - a.Deductions - a.Refund
	if tiktokFee < 0 {
		tiktokFee = 0
	}

	feeBreakdown := []map[string]interface{}{
		{"label": "tiktokCommission", "value": tiktokFee},
		{"label": "deductions", "value": a.Deductions},
		{"label": "refund", "value": a.Refund},
	}
	// filter out zero-value items to keep the chart clean
	filteredBreakdown := make([]map[string]interface{}, 0, len(feeBreakdown))
	for _, item := range feeBreakdown {
		if v, ok := item["value"].(float64); ok && v > 0 {
			filteredBreakdown = append(filteredBreakdown, item)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"gmv":            a.GMV,
		"settlement":     a.Settlement,
		"totalFees":      totalFees,
		"netProfit":      a.Settlement,
		"settlementRate": settlementRate,
		"feeBreakdown":   filteredBreakdown,
		"from":           from.Format("2006-01-02"),
		"to":             to.Format("2006-01-02"),
	})
}

// ─── GET /api/analytics/daily-metrics ────────────────────────────────────────

func AnalyticsDailyMetrics(w http.ResponseWriter, r *http.Request) {
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
	from, to := parseDateRange(r)

	if ctx.Role == auth.RoleAffiliate {
		if ctx.CompanyID == "" || ctx.UserID == "" {
			middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
			return
		}
		rows, err := fetchAffiliateRows(db, ctx.CompanyID, ctx.UserID, from, to)
		if err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		writeDailyMetrics(w, affiliateToDailyMap(rows), from, to)
		return
	}

	shopIDs, err := shopIDsForContext(db, ctx)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	rows, err := fetchImportRows(db, shopIDs, from, to)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	writeDailyMetrics(w, importToDailyMap(rows), from, to)
}

// ─── GET /api/analytics/product-metrics ──────────────────────────────────────

func AnalyticsProductMetrics(w http.ResponseWriter, r *http.Request) {
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
	from, to := parseDateRange(r)

	if ctx.Role == auth.RoleAffiliate {
		if ctx.CompanyID == "" || ctx.UserID == "" {
			middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
			return
		}
		rows, err := fetchAffiliateRows(db, ctx.CompanyID, ctx.UserID, from, to)
		if err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		writeProductMetrics(w, affiliateToSkuMap(rows), from, to)
		return
	}

	shopIDs, err := shopIDsForContext(db, ctx)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	rows, err := fetchImportRows(db, shopIDs, from, to)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	writeProductMetrics(w, importToSkuMap(rows), from, to)
}

// ─── GET /api/analytics/trends ────────────────────────────────────────────────

func AnalyticsTrends(w http.ResponseWriter, r *http.Request) {
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
	from, to := parseDateRange(r)
	period := parsePeriod(r)

	if ctx.Role == auth.RoleAffiliate {
		if ctx.CompanyID == "" || ctx.UserID == "" {
			middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
			return
		}
		rows, err := fetchAffiliateRows(db, ctx.CompanyID, ctx.UserID, from, to)
		if err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		importRows := affiliateToImportRows(rows)
		buckets := rowsToTrendBuckets(importRows, period)
		monthly := rowsToMonthlyBuckets(importRows)
		mom := trendBucketsToMomGrowth(monthly)
		yoy := trendBucketsToYoy(importRows)
		writeTrendsResponse(w, buckets, monthly, mom, yoy, from, to, period)
		return
	}

	shopIDs, err := shopIDsForContext(db, ctx)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	rows, err := fetchImportRows(db, shopIDs, from, to)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	buckets := rowsToTrendBuckets(rows, period)
	monthly := rowsToMonthlyBuckets(rows)
	mom := trendBucketsToMomGrowth(monthly)
	yoy := trendBucketsToYoy(rows)
	writeTrendsResponse(w, buckets, monthly, mom, yoy, from, to, period)
}

// affiliateToImportRows converts affiliate rows to importRow shape for trend bucketing.
func affiliateToImportRows(rows []affiliateAnalyticsRow) []importRow {
	out := make([]importRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, importRow{
			Date:    r.OrderDate,
			Revenue: r.GMV,
			Net:     r.CommissionAmount - r.IneligibleAmount,
			Quantity: r.ItemsSold,
		})
	}
	return out
}

// ─── GET /api/analytics/profitability ─────────────────────────────────────────

func AnalyticsProfitability(w http.ResponseWriter, r *http.Request) {
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
	from, to := parseDateRange(r)

	if ctx.Role == auth.RoleAffiliate {
		if ctx.CompanyID == "" || ctx.UserID == "" {
			middleware.WriteJSONError(w, middleware.ErrUnauthorized, http.StatusUnauthorized)
			return
		}
		rows, err := fetchAffiliateRows(db, ctx.CompanyID, ctx.UserID, from, to)
		if err != nil {
			middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
			return
		}
		skuMap := affiliateToSkuMap(rows)
		importRows := affiliateToImportRows(rows)
		avgMargin, marginBuckets, byCategory, marginTrend := skuMapToProfitability(skuMap, importRows, from, to)
		writeProfitabilityResponse(w, avgMargin, marginBuckets, byCategory, marginTrend, from, to)
		return
	}

	shopIDs, err := shopIDsForContext(db, ctx)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	rows, err := fetchImportRows(db, shopIDs, from, to)
	if err != nil {
		middleware.WriteJSONError(w, middleware.ErrInternal, http.StatusInternalServerError)
		return
	}
	skuMap := importToSkuMap(rows)
	avgMargin, marginBuckets, byCategory, marginTrend := skuMapToProfitability(skuMap, rows, from, to)
	writeProfitabilityResponse(w, avgMargin, marginBuckets, byCategory, marginTrend, from, to)
}

func writeProfitabilityResponse(w http.ResponseWriter, avgMargin float64, marginBuckets []map[string]interface{}, byCategory []map[string]interface{}, marginTrend []map[string]interface{}, from, to time.Time) {
	hasData := len(marginBuckets) > 0 || len(byCategory) > 0
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"avgMargin":     avgMargin,
		"marginBuckets": marginBuckets,
		"byCategory":    byCategory,
		"marginTrend":   marginTrend,
		"hasData":       hasData,
		"from":          from.Format("2006-01-02"),
		"to":            to.Format("2006-01-02"),
	})
}
