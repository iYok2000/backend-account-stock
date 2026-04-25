package calculator

// FeeRateOverrides allows dynamic fee rate overrides loaded from DB configuration.
// Rates are expressed as decimals (e.g., 0.0642 = 6.42%).
// For FIXED rates, use the rateFixed field (in Baht).
type FeeRateOverrides struct {
	CommissionRate       *float64  `json:"commissionRate,omitempty"`
	CommissionRateFixed  *float64  `json:"commissionRateFixed,omitempty"`
	CommissionRateType   FeeRateType `json:"commissionRateType,omitempty"`
	PaymentFeeRate       *float64  `json:"paymentFeeRate,omitempty"`
	PaymentFeeRateFixed  *float64  `json:"paymentFeeRateFixed,omitempty"`
	PaymentFeeRateType   FeeRateType `json:"paymentFeeRateType,omitempty"`
	CommerceGrowthRate   *float64  `json:"commerceGrowthRate,omitempty"`
	CommerceGrowthFixed  *float64  `json:"commerceGrowthRateFixed,omitempty"`
	CommerceGrowthType   FeeRateType `json:"commerceGrowthType,omitempty"`
	InfrastructureRate   *float64  `json:"infrastructureRate,omitempty"`
	InfrastructureFixed  *float64  `json:"infrastructureRateFixed,omitempty"`
	InfrastructureType   FeeRateType `json:"infrastructureType,omitempty"`
	LiveSpecialsRate     *float64  `json:"liveSpecialsRate,omitempty"`
	LiveSpecialsFixed    *float64  `json:"liveSpecialsRateFixed,omitempty"`
	LiveSpecialsType     FeeRateType `json:"liveSpecialsType,omitempty"`
	EamsRate             *float64  `json:"eamsRate,omitempty"`
	EamsRateFixed        *float64  `json:"eamsRateFixed,omitempty"`
	EamsRateType         FeeRateType `json:"eamsRateType,omitempty"`
	PreorderRate         *float64  `json:"preorderRate,omitempty"`
	PreorderRateFixed    *float64  `json:"preorderRateFixed,omitempty"`
	PreorderRateType     FeeRateType `json:"preorderRateType,omitempty"`
	EnableLiveSpecials   bool      `json:"enableLiveSpecials,omitempty"`
	EnableEams           bool      `json:"enableEams,omitempty"`
	EnablePreorder       bool      `json:"enablePreorder,omitempty"`
}

// FeeBreakdown contains the detailed fee breakdown for a sale.
type FeeBreakdown struct {
	Commission       string `json:"commission"`
	CommissionRate  string `json:"commissionRate"`
	PaymentFee      string `json:"paymentFee"`
	CommerceGrowth  string `json:"commerceGrowthFee"`
	Infrastructure  string `json:"infrastructureFee"`
	LiveSpecials    string `json:"liveSpecialsFee"`
	EamsFee         string `json:"eamsFee"`
	PreorderFee     string `json:"preorderFee"`
	TotalFees       string `json:"totalFees"`
	EffectiveRate   string `json:"effectiveRate"`
}

var zeroBreakdown = FeeBreakdown{
	Commission:     "0.00",
	CommissionRate: "0.00",
	PaymentFee:     "0.00",
	CommerceGrowth: "0.00",
	Infrastructure: "0.00",
	LiveSpecials:   "0.00",
	EamsFee:        "0.00",
	PreorderFee:    "0.00",
	TotalFees:      "0.00",
	EffectiveRate:  "0.00",
}

// getFloat returns the value or a default if nil.
func getFloat(val *float64, defaultVal float64) float64 {
	if val == nil {
		return defaultVal
	}
	return *val
}

// getRateType returns the rate type or default if empty.
func getRateType(rt FeeRateType, defaultRT FeeRateType) FeeRateType {
	if rt == "" {
		return defaultRT
	}
	return rt
}

// CalculateTikTokFees calculates TikTok Shop fees for a given sale price and product category.
//
// Fee structure:
//  1. Commission: category-based % of sale price (default 7.49%)
//  2. Payment fee: 3.21% of sale price
//  3. Commerce growth fee (optional, from overrides)
//  4. Infrastructure fee (optional, from overrides)
//  5. LIVE Specials fee (optional, from overrides)
//  6. EAMS fee (optional, from overrides)
//  7. Pre-order fee (optional, from overrides)
//
// Supports both PERCENTAGE and FIXED rate types.
//
// Returns detailed fee breakdown with all amounts as 2-decimal strings.
func CalculateTikTokFees(salePrice float64, category string, overrides *FeeRateOverrides) FeeBreakdown {
	if salePrice < 0 {
		return zeroBreakdown
	}

	// Commission
	commissionRateType := RateTypePercentage
	commissionRate := DefaultCommissionRate
	commissionRateFixed := 0.0
	if overrides != nil {
		commissionRateType = getRateType(overrides.CommissionRateType, RateTypePercentage)
		commissionRate = getFloat(overrides.CommissionRate, DefaultCommissionRate)
		if overrides.CommissionRateFixed != nil {
			commissionRateFixed = *overrides.CommissionRateFixed
		}
	}
	var commission float64
	if commissionRateType == RateTypeFixed {
		commission = round2(commissionRateFixed)
	} else {
		commission = round2(salePrice * commissionRate)
	}

	// Payment Fee
	paymentFeeRateType := RateTypePercentage
	paymentFeeRate := PaymentFeeRate
	paymentFeeRateFixed := 0.0
	if overrides != nil {
		paymentFeeRateType = getRateType(overrides.PaymentFeeRateType, RateTypePercentage)
		paymentFeeRate = getFloat(overrides.PaymentFeeRate, PaymentFeeRate)
		if overrides.PaymentFeeRateFixed != nil {
			paymentFeeRateFixed = *overrides.PaymentFeeRateFixed
		}
	}
	var paymentFee float64
	if paymentFeeRateType == RateTypeFixed {
		paymentFee = round2(paymentFeeRateFixed)
	} else {
		paymentFee = round2(salePrice * paymentFeeRate)
	}

	// Commerce Growth
	commerceGrowthRateType := RateTypePercentage
	commerceGrowthRate := 0.0
	commerceGrowthFixed := 0.0
	if overrides != nil {
		commerceGrowthRateType = getRateType(overrides.CommerceGrowthType, RateTypePercentage)
		commerceGrowthRate = getFloat(overrides.CommerceGrowthRate, 0)
		if overrides.CommerceGrowthFixed != nil {
			commerceGrowthFixed = *overrides.CommerceGrowthFixed
		}
	}
	var commerceGrowth float64
	if commerceGrowthRateType == RateTypeFixed {
		commerceGrowth = round2(commerceGrowthFixed)
	} else {
		commerceGrowth = round2(salePrice * commerceGrowthRate)
	}

	// Infrastructure
	infrastructureRateType := RateTypePercentage
	infrastructureRate := 0.0
	infrastructureFixed := 0.0
	if overrides != nil {
		infrastructureRateType = getRateType(overrides.InfrastructureType, RateTypePercentage)
		infrastructureRate = getFloat(overrides.InfrastructureRate, 0)
		if overrides.InfrastructureFixed != nil {
			infrastructureFixed = *overrides.InfrastructureFixed
		}
	}
	var infrastructure float64
	if infrastructureRateType == RateTypeFixed {
		infrastructure = round2(infrastructureFixed)
	} else {
		infrastructure = round2(salePrice * infrastructureRate)
	}

	// LIVE Specials
	liveSpecialsRateType := RateTypePercentage
	liveSpecialsRate := 0.0
	liveSpecialsFixed := 0.0
	enableLiveSpecials := false
	if overrides != nil {
		enableLiveSpecials = overrides.EnableLiveSpecials
		liveSpecialsRateType = getRateType(overrides.LiveSpecialsType, RateTypePercentage)
		liveSpecialsRate = getFloat(overrides.LiveSpecialsRate, 0)
		if overrides.LiveSpecialsFixed != nil {
			liveSpecialsFixed = *overrides.LiveSpecialsFixed
		}
	}
	var liveSpecials float64
	if enableLiveSpecials {
		if liveSpecialsRateType == RateTypeFixed {
			liveSpecials = round2(liveSpecialsFixed)
		} else {
			liveSpecials = round2(salePrice * liveSpecialsRate)
		}
	}

	// EAMS
	eamsRateType := RateTypePercentage
	eamsRate := 0.0
	eamsFixed := 0.0
	enableEams := false
	if overrides != nil {
		enableEams = overrides.EnableEams
		eamsRateType = getRateType(overrides.EamsRateType, RateTypePercentage)
		eamsRate = getFloat(overrides.EamsRate, 0)
		if overrides.EamsRateFixed != nil {
			eamsFixed = *overrides.EamsRateFixed
		}
	}
	var eams float64
	if enableEams {
		if eamsRateType == RateTypeFixed {
			eams = round2(eamsFixed)
		} else {
			eams = round2(salePrice * eamsRate)
		}
	}

	// Pre-order
	preorderRateType := RateTypePercentage
	preorderRate := 0.0
	preorderFixed := 0.0
	enablePreorder := false
	if overrides != nil {
		enablePreorder = overrides.EnablePreorder
		preorderRateType = getRateType(overrides.PreorderRateType, RateTypePercentage)
		preorderRate = getFloat(overrides.PreorderRate, PreorderFeeRate)
		if overrides.PreorderRateFixed != nil {
			preorderFixed = *overrides.PreorderRateFixed
		}
	}
	var preorder float64
	if enablePreorder {
		if preorderRateType == RateTypeFixed {
			preorder = round2(preorderFixed)
		} else {
			preorder = round2(salePrice * preorderRate)
		}
	}

	// Total fees
	totalFees := round2(commission + paymentFee + commerceGrowth + infrastructure + liveSpecials + eams + preorder)

	// Effective fee rate as percentage
	effectiveRate := 0.0
	if salePrice > 0 {
		effectiveRate = round2((totalFees / salePrice) * 100)
	}

	// Display commission rate
	displayCommissionRate := 0.0
	if commissionRateType == RateTypeFixed && salePrice > 0 && commissionRateFixed > 0 {
		displayCommissionRate = round2((commissionRateFixed / salePrice) * 100)
	} else {
		displayCommissionRate = round2(commissionRate * 100)
	}

	return FeeBreakdown{
		Commission:      toDecimal(commission),
		CommissionRate: toDecimal(displayCommissionRate),
		PaymentFee:      toDecimal(paymentFee),
		CommerceGrowth: toDecimal(commerceGrowth),
		Infrastructure: toDecimal(infrastructure),
		LiveSpecials:   toDecimal(liveSpecials),
		EamsFee:        toDecimal(eams),
		PreorderFee:    toDecimal(preorder),
		TotalFees:      toDecimal(totalFees),
		EffectiveRate:  toDecimal(effectiveRate),
	}
}
