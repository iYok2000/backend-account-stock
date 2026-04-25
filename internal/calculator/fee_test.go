package calculator

import "testing"

func ptr(f float64) *float64 { return &f }

// ─── CalculateTikTokFees ─────────────────────────────────────────────────────

func TestCalculateTikTokFees_Defaults(t *testing.T) {
	bd := CalculateTikTokFees(100.0, "", nil)
	if bd.Commission != "7.49" {
		t.Errorf("Commission = %s, want 7.49", bd.Commission)
	}
	if bd.PaymentFee != "3.21" {
		t.Errorf("PaymentFee = %s, want 3.21", bd.PaymentFee)
	}
	if bd.TotalFees != "10.70" {
		t.Errorf("TotalFees = %s, want 10.70", bd.TotalFees)
	}
	if bd.EffectiveRate != "10.70" {
		t.Errorf("EffectiveRate = %s, want 10.70", bd.EffectiveRate)
	}
	if bd.LiveSpecials != "0.00" {
		t.Errorf("LiveSpecials = %s, want 0.00", bd.LiveSpecials)
	}
	if bd.EamsFee != "0.00" {
		t.Errorf("EamsFee = %s, want 0.00", bd.EamsFee)
	}
	if bd.PreorderFee != "0.00" {
		t.Errorf("PreorderFee = %s, want 0.00", bd.PreorderFee)
	}
}

func TestCalculateTikTokFees_ZeroPrice(t *testing.T) {
	bd := CalculateTikTokFees(0, "", nil)
	if bd.TotalFees != "0.00" {
		t.Errorf("TotalFees on zero price = %s, want 0.00", bd.TotalFees)
	}
	if bd.EffectiveRate != "0.00" {
		t.Errorf("EffectiveRate on zero price = %s, want 0.00", bd.EffectiveRate)
	}
}

func TestCalculateTikTokFees_NegativePrice(t *testing.T) {
	bd := CalculateTikTokFees(-50, "", nil)
	if bd.TotalFees != "0.00" {
		t.Errorf("TotalFees on negative price = %s, want 0.00", bd.TotalFees)
	}
}

func TestCalculateTikTokFees_CustomPercentageOverrides(t *testing.T) {
	ov := &FeeRateOverrides{
		CommissionRate:     ptr(0.10),
		CommissionRateType: RateTypePercentage,
		PaymentFeeRate:     ptr(0.05),
		PaymentFeeRateType: RateTypePercentage,
	}
	bd := CalculateTikTokFees(200.0, "", ov)
	if bd.Commission != "20.00" {
		t.Errorf("Commission = %s, want 20.00", bd.Commission)
	}
	if bd.PaymentFee != "10.00" {
		t.Errorf("PaymentFee = %s, want 10.00", bd.PaymentFee)
	}
	if bd.TotalFees != "30.00" {
		t.Errorf("TotalFees = %s, want 30.00", bd.TotalFees)
	}
}

func TestCalculateTikTokFees_FixedCommission(t *testing.T) {
	ov := &FeeRateOverrides{
		CommissionRateType:  RateTypeFixed,
		CommissionRateFixed: ptr(15.00),
	}
	bd := CalculateTikTokFees(100.0, "", ov)
	if bd.Commission != "15.00" {
		t.Errorf("Commission (fixed) = %s, want 15.00", bd.Commission)
	}
}

func TestCalculateTikTokFees_FixedCommissionDisplayRate(t *testing.T) {
	// Fixed 10 on price 100 → display rate = 10.00%
	ov := &FeeRateOverrides{
		CommissionRateType:  RateTypeFixed,
		CommissionRateFixed: ptr(10.0),
	}
	bd := CalculateTikTokFees(100.0, "", ov)
	if bd.CommissionRate != "10.00" {
		t.Errorf("CommissionRate display = %s, want 10.00", bd.CommissionRate)
	}
}

func TestCalculateTikTokFees_LiveSpecialsEnabled(t *testing.T) {
	ov := &FeeRateOverrides{
		EnableLiveSpecials: true,
		LiveSpecialsRate:   ptr(0.02),
		LiveSpecialsType:   RateTypePercentage,
	}
	bd := CalculateTikTokFees(100.0, "", ov)
	if bd.LiveSpecials != "2.00" {
		t.Errorf("LiveSpecials = %s, want 2.00", bd.LiveSpecials)
	}
}

func TestCalculateTikTokFees_LiveSpecialsDisabled(t *testing.T) {
	ov := &FeeRateOverrides{
		EnableLiveSpecials: false,
		LiveSpecialsRate:   ptr(0.02),
		LiveSpecialsType:   RateTypePercentage,
	}
	bd := CalculateTikTokFees(100.0, "", ov)
	if bd.LiveSpecials != "0.00" {
		t.Errorf("LiveSpecials (disabled) = %s, want 0.00", bd.LiveSpecials)
	}
}

func TestCalculateTikTokFees_EamsEnabled(t *testing.T) {
	ov := &FeeRateOverrides{
		EnableEams:   true,
		EamsRate:     ptr(0.03),
		EamsRateType: RateTypePercentage,
	}
	bd := CalculateTikTokFees(100.0, "", ov)
	if bd.EamsFee != "3.00" {
		t.Errorf("EamsFee = %s, want 3.00", bd.EamsFee)
	}
}

func TestCalculateTikTokFees_PreorderEnabled(t *testing.T) {
	ov := &FeeRateOverrides{
		EnablePreorder:   true,
		PreorderRate:     ptr(0.02),
		PreorderRateType: RateTypePercentage,
	}
	bd := CalculateTikTokFees(100.0, "", ov)
	if bd.PreorderFee != "2.00" {
		t.Errorf("PreorderFee = %s, want 2.00", bd.PreorderFee)
	}
}

func TestCalculateTikTokFees_AllFees(t *testing.T) {
	rate := 0.01
	ov := &FeeRateOverrides{
		CommissionRate:     ptr(0.07),
		CommissionRateType: RateTypePercentage,
		PaymentFeeRate:     ptr(0.03),
		PaymentFeeRateType: RateTypePercentage,
		CommerceGrowthRate: ptr(rate),
		CommerceGrowthType: RateTypePercentage,
		InfrastructureRate: ptr(rate),
		InfrastructureType: RateTypePercentage,
		EnableLiveSpecials: true,
		LiveSpecialsRate:   ptr(rate),
		LiveSpecialsType:   RateTypePercentage,
		EnableEams:         true,
		EamsRate:           ptr(rate),
		EamsRateType:       RateTypePercentage,
		EnablePreorder:     true,
		PreorderRate:       ptr(rate),
		PreorderRateType:   RateTypePercentage,
	}
	bd := CalculateTikTokFees(100.0, "", ov)
	// 7 + 3 + 1 + 1 + 1 + 1 + 1 = 15.00
	if bd.TotalFees != "15.00" {
		t.Errorf("TotalFees (all fees) = %s, want 15.00", bd.TotalFees)
	}
	if bd.EffectiveRate != "15.00" {
		t.Errorf("EffectiveRate (all fees) = %s, want 15.00", bd.EffectiveRate)
	}
}

// ─── CalculateBatchFees ──────────────────────────────────────────────────────

func TestCalculateBatchFees_Empty(t *testing.T) {
	result := CalculateBatchFees(nil, nil)
	if result.TotalFees != "0.00" {
		t.Errorf("Empty batch TotalFees = %s, want 0.00", result.TotalFees)
	}
	if result.ItemCount != 0 {
		t.Errorf("Empty batch ItemCount = %d, want 0", result.ItemCount)
	}
}

func TestCalculateBatchFees_SingleItem(t *testing.T) {
	items := []BatchItem{
		{SalePrice: 100.0, Category: "", Quantity: 1},
	}
	result := CalculateBatchFees(items, nil)
	if result.ItemCount != 1 {
		t.Errorf("ItemCount = %d, want 1", result.ItemCount)
	}
	if result.TotalFees != "10.70" {
		t.Errorf("TotalFees = %s, want 10.70", result.TotalFees)
	}
}

func TestCalculateBatchFees_MultipleItems(t *testing.T) {
	items := []BatchItem{
		{SalePrice: 100.0, Category: "", Quantity: 2},
		{SalePrice: 200.0, Category: "", Quantity: 1},
	}
	result := CalculateBatchFees(items, nil)
	if result.ItemCount != 3 {
		t.Errorf("ItemCount = %d, want 3", result.ItemCount)
	}
	// item1: 2 × (100×0.0749 + 100×0.0321) = 2 × 10.70 = 21.40
	// item2: 1 × (200×0.0749 + 200×0.0321) = 1 × 21.40 = 21.40
	// total: 42.80
	if result.TotalFees != "42.80" {
		t.Errorf("TotalFees = %s, want 42.80", result.TotalFees)
	}
}

func TestCalculateBatchFees_WithOverrides(t *testing.T) {
	ov := &FeeRateOverrides{
		CommissionRate:     ptr(0.10),
		CommissionRateType: RateTypePercentage,
		PaymentFeeRate:     ptr(0.0),
		PaymentFeeRateType: RateTypePercentage,
	}
	items := []BatchItem{
		{SalePrice: 100.0, Quantity: 1},
	}
	result := CalculateBatchFees(items, ov)
	if result.TotalFees != "10.00" {
		t.Errorf("TotalFees with overrides = %s, want 10.00", result.TotalFees)
	}
}

// ─── Internal helpers ────────────────────────────────────────────────────────

func TestRound2(t *testing.T) {
	cases := []struct {
		input float64
		want  float64
	}{
		{1.005, 1.00}, // IEEE 754: 1.005*100 = 100.4999... → rounds to 100
		{1.006, 1.01},
		{1.004, 1.00},
		{7.4900000001, 7.49},
		{0.0, 0.0},
	}
	for _, tc := range cases {
		got := round2(tc.input)
		if got != tc.want {
			t.Errorf("round2(%v) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestGetFloat_Nil(t *testing.T) {
	got := getFloat(nil, 3.14)
	if got != 3.14 {
		t.Errorf("getFloat(nil, 3.14) = %v, want 3.14", got)
	}
}

func TestGetFloat_NotNil(t *testing.T) {
	v := 9.99
	got := getFloat(&v, 0)
	if got != 9.99 {
		t.Errorf("getFloat(&9.99, 0) = %v, want 9.99", got)
	}
}

func TestGetRateType_Empty(t *testing.T) {
	got := getRateType("", RateTypePercentage)
	if got != RateTypePercentage {
		t.Errorf("getRateType('', PERCENTAGE) = %v, want PERCENTAGE", got)
	}
}

func TestGetRateType_Set(t *testing.T) {
	got := getRateType(RateTypeFixed, RateTypePercentage)
	if got != RateTypeFixed {
		t.Errorf("getRateType(FIXED, PERCENTAGE) = %v, want FIXED", got)
	}
}
