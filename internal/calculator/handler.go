package calculator

import (
	"encoding/json"
	"net/http"

	"account-stock-be/internal/middleware"
)

// CalculateFeesRequest for POST /api/calculator/fees.
type CalculateFeesRequest struct {
	SalePrice float64             `json:"salePrice"`
	Category  string             `json:"category"`
	Overrides *FeeRateOverrides  `json:"overrides,omitempty"`
}

// CalculateFees handles POST /api/calculator/fees.
// Calculates TikTok Shop fees for a given sale price and optional fee overrides.
func CalculateFees(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req CalculateFeesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
		return
	}

	if req.SalePrice < 0 {
		middleware.WriteJSONError(w, "sale price cannot be negative", http.StatusBadRequest)
		return
	}

	breakdown := CalculateTikTokFees(req.SalePrice, req.Category, req.Overrides)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"salePrice": req.SalePrice,
		"category":  req.Category,
		"fees":     breakdown,
	})
}

// CalculateBatchFeesRequest for POST /api/calculator/fees/batch.
type CalculateBatchFeesRequest struct {
	Items    []BatchItem        `json:"items"`
	Overrides *FeeRateOverrides `json:"overrides,omitempty"`
}

// CalculateBatchFeesHandler handles POST /api/calculator/fees/batch.
func CalculateBatchFeesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req CalculateBatchFeesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		middleware.WriteJSONError(w, "items required", http.StatusBadRequest)
		return
	}

	result := CalculateBatchFees(req.Items, req.Overrides)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":  req.Items,
		"result": result,
	})
}
