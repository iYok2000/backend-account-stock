package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	"account-stock-be/internal/auth"
	"account-stock-be/internal/database"
	"account-stock-be/internal/handler"
	"account-stock-be/internal/middleware"
	"account-stock-be/internal/rbac"
)

func init() {
	// Force IPv4 only (Railway/Supabase connectivity issue with IPv6)
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "tcp4", address)
		},
	}
}

func main() {
	// Production: refuse to start with dev JWT secret
	if os.Getenv("APP_ENV") == "production" {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" || secret == "dev-secret-change-in-production" {
			log.Fatal("production requires JWT_SECRET to be set and not the dev default")
		}
	}
	jwtCfg := auth.DefaultJWTConfig()

	// Connect to DB (DefaultConfig has fallback for local dev)
	dbCfg := database.DefaultConfig()
	if _, err := database.Open(dbCfg); err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()

	mux := http.NewServeMux()

	// Public (no auth)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Consent / legal acknowledgment (PDPA)
	mux.HandleFunc("/api/consent/pdpa", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.GetPDPAConsent(w, r)
		case http.MethodPost:
			handler.PostPDPAConsent(w, r)
		default:
			middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		}
	})

	// API: auth — POST /api/auth/login (no JWT); GET /api/auth/me (JWT required)
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		handler.Login(w, r, jwtCfg)
	})
	apiAuth := http.NewServeMux()
	apiAuth.HandleFunc("/me", middleware.RequireAuthContext(handler.Me))
	mux.Handle("/api/auth/", http.StripPrefix("/api/auth", middleware.Auth(jwtCfg)(apiAuth)))

	// API: inventory import (SKU/day source of truth)
	invImportHandler := http.HandlerFunc(handler.ImportInventory)
	invImportChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermInventoryCreate)(middleware.Tenant(invImportHandler)))
	mux.Handle("/api/inventory/import", invImportChain)

	// API: inventory list
	invListHandler := http.HandlerFunc(handler.InventoryList)
	invListChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermInventoryRead)(middleware.Tenant(invListHandler)))
	mux.Handle("/api/inventory", invListChain)

	// API: inventory summary
	invSummaryHandler := http.HandlerFunc(handler.InventorySummary)
	invSummaryChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermInventoryRead)(middleware.Tenant(invSummaryHandler)))
	mux.Handle("/api/inventory/summary", invSummaryChain)

	// API: users — Auth then RequirePermission(users:read)
	usersHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
			return
		}
		handler.UsersList(w, r)
	})
	usersChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermUsersRead)(middleware.Tenant(usersHandler)))
	mux.Handle("/api/users", usersChain)

	// API: shops — POST /api/shops (Root only), GET/PATCH /api/shops/me (SuperAdmin), POST /api/shops/me/members (SuperAdmin)
	shopsCreateChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermShopsCreate)(middleware.Tenant(http.HandlerFunc(handler.CreateShops))))
	mux.Handle("/api/shops", shopsCreateChain)

	shopsMeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.GetShopsMe(w, r)
		case http.MethodPatch, http.MethodPut:
			handler.PatchShopsMe(w, r)
		default:
			middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		}
	})
	shopsMeChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermShopsUpdate)(middleware.Tenant(shopsMeHandler)))
	mux.Handle("/api/shops/me", shopsMeChain)

	shopsMeMembersHandler := http.HandlerFunc(handler.ShopsMeMembers)
	shopsMeMembersChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermUsersCreate)(middleware.Tenant(shopsMeMembersHandler)))
	mux.Handle("/api/shops/me/members", shopsMeMembersChain)

	// API: self (PATCH update display/password, DELETE self)
	selfHandler := http.HandlerFunc(handler.Self)
	selfChain := middleware.Auth(jwtCfg)(middleware.Tenant(selfHandler))
	mux.Handle("/api/users/me", selfChain)

	// API: dashboard (overview + revenue + low stock)
	dashboardOverview := http.HandlerFunc(handler.DashboardOverview)
	dashboardOverviewChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermDashboardRead)(middleware.Tenant(dashboardOverview)))
	mux.Handle("/api/dashboard/overview", dashboardOverviewChain)

	dashboardRevenue := http.HandlerFunc(handler.DashboardRevenue7d)
	dashboardRevenueChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermDashboardRead)(middleware.Tenant(dashboardRevenue)))
	mux.Handle("/api/dashboard/revenue-7d", dashboardRevenueChain)

	dashboardLowStock := http.HandlerFunc(handler.DashboardLowStock)
	dashboardLowStockChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermDashboardRead)(middleware.Tenant(dashboardLowStock)))
	mux.Handle("/api/dashboard/low-stock", dashboardLowStockChain)

	dashboardKPIs := http.HandlerFunc(handler.DashboardKPIs)
	dashboardKPIsChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermDashboardRead)(middleware.Tenant(dashboardKPIs)))
	mux.Handle("/api/dashboard/kpis", dashboardKPIsChain)

	// API: affiliate import (Affiliate uploads) — use inventory:create permission
	affiliateImportHandler := http.HandlerFunc(handler.AffiliateImport)
	affiliateImportChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermInventoryCreate)(middleware.Tenant(affiliateImportHandler)))
	mux.Handle("/api/affiliate/import", affiliateImportChain)

	// API: analytics (reconciliation, daily metrics, product metrics)
	analyticsRecon := http.HandlerFunc(handler.AnalyticsReconciliation)
	analyticsReconChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermAnalyticsRead)(middleware.Tenant(analyticsRecon)))
	mux.Handle("/api/analytics/reconciliation", analyticsReconChain)

	analyticsDaily := http.HandlerFunc(handler.AnalyticsDailyMetrics)
	analyticsDailyChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermAnalyticsRead)(middleware.Tenant(analyticsDaily)))
	mux.Handle("/api/analytics/daily-metrics", analyticsDailyChain)

	analyticsProducts := http.HandlerFunc(handler.AnalyticsProductMetrics)
	analyticsProductsChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermAnalyticsRead)(middleware.Tenant(analyticsProducts)))
	mux.Handle("/api/analytics/product-metrics", analyticsProductsChain)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, middleware.CORS(mux)); err != nil {
		log.Fatal(err)
	}
}
