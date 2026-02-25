package handler

import (
	"encoding/json"
	"net/http"
)

// ImportOrderTransactionResponse stub response until import is persisted.
type ImportOrderTransactionResponse struct {
	OK bool `json:"ok"`
}

// ImportOrderTransaction handles POST /api/import/order-transaction.
// Accepts JSON body (tier, summary, daily or items); returns 200 with {"ok": true}.
// Persistence can be added later.
func ImportOrderTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// middleware would typically handle this; allow POST only
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// Decode to validate; ignore body for now (stub)
	var body struct {
		Tier    string `json:"tier"`
		Summary interface{} `json:"summary"`
		Daily   interface{} `json:"daily,omitempty"`
		Items   interface{} `json:"items,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ImportOrderTransactionResponse{OK: true})
}
