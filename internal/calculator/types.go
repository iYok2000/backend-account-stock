package calculator

import (
	"fmt"
	"math"
)

// FeeRateType represents whether a fee is a percentage or fixed amount.
type FeeRateType string

const (
	RateTypePercentage FeeRateType = "PERCENTAGE"
	RateTypeFixed      FeeRateType = "FIXED"
)

// Default TikTok Shop fee rates (Thailand).
const (
	DefaultCommissionRate = 0.0749  // 7.49%
	PaymentFeeRate        = 0.0321  // 3.21%
	PreorderFeeRate       = 0.02    // 2.0%
)

// round2 rounds a float64 to 2 decimal places using "round half up" (banker's rounding avoidance).
func round2(n float64) float64 {
	return math.Round(n*100) / 100
}

// toDecimal formats a float64 as a string with 2 decimal places.
func toDecimal(n float64) string {
	return fmt.Sprintf("%.2f", n)
}
