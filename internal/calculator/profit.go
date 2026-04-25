package calculator

// BatchItem represents a single item in a batch fee calculation.
type BatchItem struct {
	SalePrice float64 `json:"salePrice"`
	Category  string  `json:"category"`
	Quantity  int     `json:"quantity"`
}

// BatchResult is the result of a batch fee calculation.
type BatchResult struct {
	FeeBreakdown
	ItemCount int `json:"itemCount"`
}

// CalculateBatchFees calculates fees for a batch of items.
// Useful for computing total fees across an order or product aggregate.
func CalculateBatchFees(items []BatchItem, overrides *FeeRateOverrides) BatchResult {
	if len(items) == 0 {
		return BatchResult{
			FeeBreakdown: zeroBreakdown,
			ItemCount:   0,
		}
	}

	var totalCommission, totalPaymentFee, totalCommerceGrowth float64
	var totalInfrastructure, totalLiveSpecials, totalEams, totalPreorder float64
	var totalSalePrice float64
	itemCount := 0

	// Get rate types and defaults
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

	for _, item := range items {
		lineTotal := item.SalePrice * float64(item.Quantity)
		itemCount += item.Quantity

		// Commission
		if commissionRateType == RateTypeFixed {
			totalCommission += round2(commissionRateFixed * float64(item.Quantity))
		} else {
			totalCommission += round2(lineTotal * commissionRate)
		}

		// Payment Fee
		if paymentFeeRateType == RateTypeFixed {
			totalPaymentFee += round2(paymentFeeRateFixed * float64(item.Quantity))
		} else {
			totalPaymentFee += round2(lineTotal * paymentFeeRate)
		}

		// Commerce Growth
		if commerceGrowthRateType == RateTypeFixed {
			totalCommerceGrowth += round2(commerceGrowthFixed * float64(item.Quantity))
		} else {
			totalCommerceGrowth += round2(lineTotal * commerceGrowthRate)
		}

		// Infrastructure
		if infrastructureRateType == RateTypeFixed {
			totalInfrastructure += round2(infrastructureFixed * float64(item.Quantity))
		} else {
			totalInfrastructure += round2(lineTotal * infrastructureRate)
		}

		// LIVE Specials
		if enableLiveSpecials {
			if liveSpecialsRateType == RateTypeFixed {
				totalLiveSpecials += round2(liveSpecialsFixed * float64(item.Quantity))
			} else {
				totalLiveSpecials += round2(lineTotal * liveSpecialsRate)
			}
		}

		// EAMS
		if enableEams {
			if eamsRateType == RateTypeFixed {
				totalEams += round2(eamsFixed * float64(item.Quantity))
			} else {
				totalEams += round2(lineTotal * eamsRate)
			}
		}

		// Pre-order
		if enablePreorder {
			if preorderRateType == RateTypeFixed {
				totalPreorder += round2(preorderFixed * float64(item.Quantity))
			} else {
				totalPreorder += round2(lineTotal * preorderRate)
			}
		}

		totalSalePrice += lineTotal
	}

	totalFees := round2(totalCommission + totalPaymentFee + totalCommerceGrowth + totalInfrastructure + totalLiveSpecials + totalEams + totalPreorder)
	effectiveRate := 0.0
	if totalSalePrice > 0 {
		effectiveRate = round2((totalFees / totalSalePrice) * 100)
	}

	displayCommissionRate := 0.0
	if commissionRateType == RateTypeFixed && totalSalePrice > 0 && commissionRateFixed > 0 {
		displayCommissionRate = round2((totalCommission / totalSalePrice) * 100)
	} else {
		displayCommissionRate = round2(commissionRate * 100)
	}

	return BatchResult{
		FeeBreakdown: FeeBreakdown{
			Commission:      toDecimal(totalCommission),
			CommissionRate: toDecimal(displayCommissionRate),
			PaymentFee:      toDecimal(totalPaymentFee),
			CommerceGrowth: toDecimal(totalCommerceGrowth),
			Infrastructure: toDecimal(totalInfrastructure),
			LiveSpecials:   toDecimal(totalLiveSpecials),
			EamsFee:        toDecimal(totalEams),
			PreorderFee:    toDecimal(totalPreorder),
			TotalFees:      toDecimal(totalFees),
			EffectiveRate:  toDecimal(effectiveRate),
		},
		ItemCount: itemCount,
	}
}
