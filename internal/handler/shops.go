package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"account-stock-be/internal/auth"
	"account-stock-be/internal/database"
	"account-stock-be/internal/middleware"
	"account-stock-be/internal/model"

	"gorm.io/gorm"
)

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateShopsRequest body for POST /api/shops (Root only).
type CreateShopsRequest struct {
	Name    string             `json:"name"`
	Members []CreateShopMember `json:"members"`
}

type CreateShopMember struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// CreateShops handles POST /api/shops. Root only; creates shop + users (at least one SuperAdmin).
func CreateShops(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ctx := middleware.GetContext(r.Context())
	if ctx == nil || ctx.Role != auth.RoleRoot {
		middleware.WriteJSONError(w, middleware.ErrForbidden, http.StatusForbidden)
		return
	}
	var body CreateShopsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		middleware.WriteJSONError(w, "name required", http.StatusBadRequest)
		return
	}
	// SHOPS_AND_ROLES_SPEC §3: must have at least 1 member who is SuperAdmin
	if len(body.Members) == 0 {
		middleware.WriteJSONError(w, "at least one member (SuperAdmin) is required", http.StatusBadRequest)
		return
	}
	hasSuperAdmin := false
	roleAllow := map[string]bool{"SuperAdmin": true, "Admin": true, "Affiliate": true}
	for _, m := range body.Members {
		if !roleAllow[m.Role] {
			middleware.WriteJSONError(w, "invalid role", http.StatusBadRequest)
			return
		}
		em := strings.TrimSpace(m.Email)
		if em == "" || m.Password == "" {
			middleware.WriteJSONError(w, "email and password required for each member", http.StatusBadRequest)
			return
		}
		if m.Role == "SuperAdmin" {
			hasSuperAdmin = true
		}
	}
	if !hasSuperAdmin {
		middleware.WriteJSONError(w, "at least one member must be SuperAdmin", http.StatusBadRequest)
		return
	}

	db := database.DB()
	if db == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	companyID := newID()
	shopID := newID()
	// transaction to keep company, shop, users consistent
	if err := db.Transaction(func(tx *gorm.DB) error {
		company := model.Company{ID: companyID, Name: name}
		if err := tx.Create(&company).Error; err != nil {
			return err
		}
		shop := model.Shop{ID: shopID, CompanyID: companyID, Name: name}
		if err := tx.Create(&shop).Error; err != nil {
			return err
		}
		for _, m := range body.Members {
			hash, err := auth.HashPassword(m.Password)
			if err != nil {
				return err
			}
			uid := newID()
			u := model.User{
				ID:           uid,
				Email:        strings.TrimSpace(m.Email),
				PasswordHash: hash,
				Role:         m.Role,
				Tier:         "free",
				ShopID:       &shopID,
				CompanyID:    companyID,
			}
			if err := tx.Create(&u).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			middleware.WriteJSONError(w, "email already exists", http.StatusBadRequest)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(struct {
		ID        string `json:"id"`
		CompanyID string `json:"company_id"`
	}{ID: shopID, CompanyID: companyID})
}

// ShopsMeResponse for GET /api/shops/me.
type ShopsMeResponse struct {
	Name    string       `json:"name"`
	Members []MemberItem `json:"members"`
}

type MemberItem struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// ensureRootDefaultShop creates default company/shop for Root (YPC / YP-SHOP) if missing.
func ensureRootDefaultShop(db *gorm.DB) (companyID, shopID string, err error) {
	companyID = defaultRootCompanyID
	shopID = defaultRootShopID
	// company
	var company model.Company
	if err = db.Where("id = ?", companyID).First(&company).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return "", "", err
		}
		if err = db.Create(&model.Company{ID: companyID, Name: defaultRootShopName}).Error; err != nil {
			return "", "", err
		}
	}
	// shop
	var shop model.Shop
	if err = db.Where("id = ? AND company_id = ?", shopID, companyID).First(&shop).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return "", "", err
		}
		if err = db.Create(&model.Shop{ID: shopID, CompanyID: companyID, Name: defaultRootShopName}).Error; err != nil {
			return "", "", err
		}
	}
	return companyID, shopID, nil
}

// GetShopsMe handles GET /api/shops/me. SuperAdmin of that shop (users:read).
func GetShopsMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ctx := middleware.GetContext(r.Context())
	if ctx == nil {
		middleware.WriteJSONError(w, middleware.ErrForbidden, http.StatusForbidden)
		return
	}
	db := database.DB()
	if db == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	shopID := ctx.ShopID
	companyID := ctx.CompanyID
	// Root: always ensure default shop/company exists (even if token already has shop_id)
	if ctx.Role == auth.RoleRoot {
		var err error
		companyID, shopID, err = ensureRootDefaultShop(db)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	var shop model.Shop
	if err := db.Where("id = ? AND company_id = ?", shopID, companyID).First(&shop).Error; err != nil {
		middleware.WriteJSONError(w, "shop not found", http.StatusNotFound)
		return
	}
	var users []model.User
	if err := db.Where("shop_id = ? AND company_id = ?", shopID, companyID).Find(&users).Error; err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	members := make([]MemberItem, 0, len(users))
	for _, u := range users {
		members = append(members, MemberItem{ID: u.ID, Email: u.Email, Role: u.Role})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ShopsMeResponse{Name: shop.Name, Members: members})
}

// PatchShopsMeRequest for PATCH /api/shops/me.
type PatchShopsMeRequest struct {
	Name string `json:"name"`
}

// PatchShopsMe handles PATCH /api/shops/me. SuperAdmin only (shops:update).
func PatchShopsMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	ctx := middleware.GetContext(r.Context())
	if ctx == nil {
		middleware.WriteJSONError(w, middleware.ErrForbidden, http.StatusForbidden)
		return
	}
	var body PatchShopsMeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		middleware.WriteJSONError(w, "name required", http.StatusBadRequest)
		return
	}
	db := database.DB()
	if db == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	shopID := ctx.ShopID
	companyID := ctx.CompanyID
	if ctx.Role == auth.RoleRoot {
		var err error
		companyID, shopID, err = ensureRootDefaultShop(db)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	if err := db.Model(&model.Shop{}).
		Where("id = ? AND company_id = ?", shopID, companyID).
		Update("name", name).Error; err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// PostShopsMeMembersRequest for POST /api/shops/me/members.
type PostShopsMeMembersRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// ShopsMeMembers handles POST/PATCH/DELETE /api/shops/me/members.
// SuperAdmin only (users:create). Role must be Admin or Affiliate when creating/updating.
func ShopsMeMembers(w http.ResponseWriter, r *http.Request) {
	ctx := middleware.GetContext(r.Context())
	if ctx == nil {
		middleware.WriteJSONError(w, middleware.ErrForbidden, http.StatusForbidden)
		return
	}
	db := database.DB()
	if db == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	shopID := ctx.ShopID
	companyID := ctx.CompanyID
	if ctx.Role == auth.RoleRoot {
		var err error
		companyID, shopID, err = ensureRootDefaultShop(db)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	switch r.Method {
	case http.MethodPost:
		var body PostShopsMeMembersRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
			return
		}
		email := strings.TrimSpace(body.Email)
		if email == "" || body.Password == "" {
			middleware.WriteJSONError(w, "email and password required", http.StatusBadRequest)
			return
		}
		if body.Role != "Admin" && body.Role != "Affiliate" {
			middleware.WriteJSONError(w, "role must be Admin or Affiliate", http.StatusBadRequest)
			return
		}
		hash, err := auth.HashPassword(body.Password)
		if err != nil {
			middleware.WriteJSONErrorMsg(w, err.Error(), http.StatusInternalServerError)
			return
		}
		uid := newID()
		u := model.User{
			ID:           uid,
			Email:        email,
			PasswordHash: hash,
			Role:         body.Role,
			Tier:         "free",
			ShopID:       &shopID,
			CompanyID:    companyID,
		}
		if err := db.Create(&u).Error; err != nil {
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				middleware.WriteJSONError(w, "email already exists", http.StatusBadRequest)
				return
			}
			middleware.WriteJSONErrorMsg(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(MemberItem{ID: u.ID, Email: u.Email, Role: u.Role})

	case http.MethodPatch:
		var body struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
			return
		}
		if body.ID == "" {
			middleware.WriteJSONError(w, "id required", http.StatusBadRequest)
			return
		}
		if body.Role != "Admin" && body.Role != "Affiliate" {
			middleware.WriteJSONError(w, "role must be Admin or Affiliate", http.StatusBadRequest)
			return
		}
		if err := db.Model(&model.User{}).
			Where("id = ? AND shop_id = ? AND company_id = ?", body.ID, shopID, companyID).
			Update("role", body.Role).Error; err != nil {
			middleware.WriteJSONErrorMsg(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		var body struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			middleware.WriteJSONError(w, middleware.ErrInvalidJSON, http.StatusBadRequest)
			return
		}
		if body.ID == "" {
			middleware.WriteJSONError(w, "id required", http.StatusBadRequest)
			return
		}
		if err := db.Where("id = ? AND shop_id = ? AND company_id = ?", body.ID, shopID, companyID).
			Delete(&model.User{}).Error; err != nil {
			middleware.WriteJSONErrorMsg(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}
