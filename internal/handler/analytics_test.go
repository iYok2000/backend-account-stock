package handler

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

// parseDateRange

func TestParseDateRange_Defaults(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	start, end := parseDateRange(r)
	now := time.Now().In(tzBangkok).Truncate(24 * time.Hour)
	if !end.Equal(now) {
		t.Errorf("default end = %v, want %v", end, now)
	}
	wantStart := now.AddDate(0, 0, -29)
	if !start.Equal(wantStart) {
		t.Errorf("default start = %v, want %v", start, wantStart)
	}
}

func TestParseDateRange_ExplicitDates(t *testing.T) {
	r := httptest.NewRequest("GET", "/?from=2026-01-01&to=2026-01-31", nil)
	start, end := parseDateRange(r)
	if start.Year() != 2026 || start.Month() != 1 || start.Day() != 1 {
		t.Errorf("start = %v, want 2026-01-01", start)
	}
	if end.Year() != 2026 || end.Month() != 1 || end.Day() != 31 {
		t.Errorf("end = %v, want 2026-01-31", end)
	}
}

func TestParseDateRange_SwapsWhenReversed(t *testing.T) {
	r := httptest.NewRequest("GET", "/?from=2026-03-31&to=2026-01-01", nil)
	start, end := parseDateRange(r)
	if start.After(end) {
		t.Errorf("start (%v) should not be after end (%v)", start, end)
	}
}

func TestParseDateRange_CapsAtMaxDays(t *testing.T) {
	r := httptest.NewRequest("GET", "/?from=2020-01-01&to=2026-01-01", nil)
	start, end := parseDateRange(r)
	days := end.Sub(start).Hours() / 24
	if days > float64(maxDateRangeDays) {
		t.Errorf("range = %.0f days, want <= %d", days, maxDateRangeDays)
	}
}

func TestParseDateRange_InvalidFromIgnored(t *testing.T) {
	// invalid from → start uses default (today-29); valid far-future to is used
	r := httptest.NewRequest("GET", "/?from=not-a-date&to=2026-12-31", nil)
	start, end := parseDateRange(r)
	if end.Year() != 2026 || end.Month() != 12 || end.Day() != 31 {
		t.Errorf("end = %v, want 2026-12-31", end)
	}
	if start.After(end) {
		t.Errorf("start (%v) should not be after end (%v)", start, end)
	}
}

// parsePeriod

func TestParsePeriod_Default(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if got := parsePeriod(r); got != "monthly" {
		t.Errorf("parsePeriod() = %s, want monthly", got)
	}
}

func TestParsePeriod_Weekly(t *testing.T) {
	r := httptest.NewRequest("GET", "/?period=weekly", nil)
	if got := parsePeriod(r); got != "weekly" {
		t.Errorf("parsePeriod(weekly) = %s, want weekly", got)
	}
}

func TestParsePeriod_InvalidFallback(t *testing.T) {
	r := httptest.NewRequest("GET", "/?period=daily", nil)
	if got := parsePeriod(r); got != "monthly" {
		t.Errorf("parsePeriod(invalid) = %s, want monthly", got)
	}
}

// aggregateImport

func TestAggregateImport_Empty(t *testing.T) {
	a := aggregateImport(nil)
	if a.GMV != 0 || a.Settlement != 0 || a.Deductions != 0 || a.Refund != 0 {
		t.Errorf("aggregateImport(nil) non-zero: %+v", a)
	}
}

func TestAggregateImport_Sums(t *testing.T) {
	rows := []importRow{
		{Revenue: 100, Net: 80, Deductions: 15, Refund: 5},
		{Revenue: 200, Net: 160, Deductions: 30, Refund: 10},
	}
	a := aggregateImport(rows)
	if a.GMV != 300 {
		t.Errorf("GMV = %v, want 300", a.GMV)
	}
	if a.Settlement != 240 {
		t.Errorf("Settlement = %v, want 240", a.Settlement)
	}
	if a.Deductions != 45 {
		t.Errorf("Deductions = %v, want 45", a.Deductions)
	}
	if a.Refund != 15 {
		t.Errorf("Refund = %v, want 15", a.Refund)
	}
}

// aggregateAffiliate

func TestAggregateAffiliate_Empty(t *testing.T) {
	a := aggregateAffiliate(nil)
	if a.GMV != 0 || a.Earned != 0 || a.Ineligible != 0 {
		t.Errorf("aggregateAffiliate(nil) non-zero: %+v", a)
	}
}

func TestAggregateAffiliate_Sums(t *testing.T) {
	rows := []affiliateAnalyticsRow{
		{GMV: 500, CommissionAmount: 50, IneligibleAmount: 10},
		{GMV: 300, CommissionAmount: 30, IneligibleAmount: 5},
	}
	a := aggregateAffiliate(rows)
	if a.GMV != 800 {
		t.Errorf("GMV = %v, want 800", a.GMV)
	}
	if a.Earned != 80 {
		t.Errorf("Earned = %v, want 80", a.Earned)
	}
	if a.Ineligible != 15 {
		t.Errorf("Ineligible = %v, want 15", a.Ineligible)
	}
}

// importToDailyMap

func TestImportToDailyMap_GroupsByDate(t *testing.T) {
	day1, _ := time.Parse("2006-01-02", "2026-01-01")
	day2, _ := time.Parse("2006-01-02", "2026-01-02")
	rows := []importRow{
		{Date: day1, Revenue: 100, Net: 80},
		{Date: day1, Revenue: 50, Net: 40},
		{Date: day2, Revenue: 200, Net: 160},
	}
	m := importToDailyMap(rows)
	if len(m) != 2 {
		t.Fatalf("map len = %d, want 2", len(m))
	}
	d1 := m["2026-01-01"]
	if d1 == nil {
		t.Fatal("expected entry for 2026-01-01")
	}
	if d1.revenue != 150 {
		t.Errorf("day1 revenue = %v, want 150", d1.revenue)
	}
	if d1.profit != 120 {
		t.Errorf("day1 profit = %v, want 120", d1.profit)
	}
}

// importToSkuMap

func TestImportToSkuMap_GroupsBySku(t *testing.T) {
	day, _ := time.Parse("2006-01-02", "2026-01-01")
	rows := []importRow{
		{SKUID: "sku1", ProductName: "Product A", Date: day, Quantity: 2, Revenue: 100, Net: 80},
		{SKUID: "sku1", ProductName: "Product A", Date: day, Quantity: 3, Revenue: 150, Net: 120},
		{SKUID: "sku2", ProductName: "Product B", Date: day, Quantity: 1, Revenue: 50, Net: 40},
	}
	m := importToSkuMap(rows)
	if len(m) != 2 {
		t.Fatalf("map len = %d, want 2", len(m))
	}
	s1 := m["sku1"]
	if s1 == nil {
		t.Fatal("expected entry for sku1")
	}
	if s1.quantity != 5 {
		t.Errorf("sku1 quantity = %v, want 5", s1.quantity)
	}
	if s1.revenue != 250 {
		t.Errorf("sku1 revenue = %v, want 250", s1.revenue)
	}
	if s1.profit != 200 {
		t.Errorf("sku1 profit = %v, want 200", s1.profit)
	}
}

func TestImportToSkuMap_FallsBackToSkuID(t *testing.T) {
	day, _ := time.Parse("2006-01-02", "2026-01-01")
	rows := []importRow{
		{SKUID: "sku-no-name", ProductName: "", Date: day, Quantity: 1, Revenue: 10, Net: 8},
	}
	m := importToSkuMap(rows)
	s := m["sku-no-name"]
	if s == nil {
		t.Fatal("expected entry for sku-no-name")
	}
	if s.name != "sku-no-name" {
		t.Errorf("name = %s, want sku-no-name", s.name)
	}
}

// shopIDsForContext (nil path, no DB needed)

func TestShopIDsForContext_NilContext(t *testing.T) {
	ids, err := shopIDsForContext(nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ids != nil {
		t.Errorf("expected nil ids, got %v", ids)
	}
}

// Sprintf format sanity check

func TestSprintf2DP(t *testing.T) {
	cases := []struct {
		val  float64
		want string
	}{
		{7.49, "7.49"},
		{0.0, "0.00"},
		{100.0, "100.00"},
		{3.21999, "3.22"},
	}
	for _, tc := range cases {
		got := fmt.Sprintf("%.2f", tc.val)
		if got != tc.want {
			t.Errorf("Sprintf(%v) = %s, want %s", tc.val, got, tc.want)
		}
	}
}
