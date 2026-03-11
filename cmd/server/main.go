package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"account-stock-be/internal/auth"
	"account-stock-be/internal/database"
	"account-stock-be/internal/handler"
	"account-stock-be/internal/middleware"
	"account-stock-be/internal/rbac"
)

// func init() {
// 	// Force IPv4 (Railway + Supabase issue)
// 	net.DefaultResolver = &net.Resolver{
// 		PreferGo: true,
// 		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
// 			d := net.Dialer{}
// 			return d.DialContext(ctx, "tcp4", address)
// 		},
// 	}
// }

func main() {

	// -----------------------------
	// ENV Validation
	// -----------------------------

	appEnv := os.Getenv("APP_ENV")
	log.Println("APP_ENV:", appEnv)

	if appEnv == "production" {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" || secret == "dev-secret-change-in-production" {
			log.Fatal("production requires JWT_SECRET to be set and not the dev default")
		}
	}

	// -----------------------------
	// JWT Config
	// -----------------------------

	jwtCfg := auth.DefaultJWTConfig()

	// -----------------------------
	// Database Connect (Retry)
	// -----------------------------

	dbCfg := database.DefaultConfig()

	connectDB(dbCfg)

	defer database.Close()

	// -----------------------------
	// Router
	// -----------------------------

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// -----------------------------
	// Consent (PDPA)
	// -----------------------------

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

	// -----------------------------
	// Auth
	// -----------------------------

	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		handler.Login(w, r, jwtCfg)
	})

	apiAuth := http.NewServeMux()
	apiAuth.HandleFunc("/me", middleware.RequireAuthContext(handler.Me))

	mux.Handle(
		"/api/auth/",
		http.StripPrefix("/api/auth", middleware.Auth(jwtCfg)(middleware.Tenant(apiAuth))),
	)

	// -----------------------------
	// Inventory
	// -----------------------------

	invImportHandler := http.HandlerFunc(handler.ImportInventory)
	invImportChain :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermInventoryCreate)(invImportHandler),
			),
		)

	mux.Handle("/api/inventory/import", invImportChain)

	invListHandler := http.HandlerFunc(handler.InventoryList)
	invListChain :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermInventoryRead)(invListHandler),
			),
		)

	mux.Handle("/api/inventory", invListChain)

	invSummaryHandler := http.HandlerFunc(handler.InventorySummary)
	invSummaryChain :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermInventoryRead)(invSummaryHandler),
			),
		)

	mux.Handle("/api/inventory/summary", invSummaryChain)

	// -----------------------------
	// Users
	// -----------------------------

	usersHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodGet {
			middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
			return
		}

		handler.UsersList(w, r)

	})

	usersChain :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermUsersRead)(usersHandler),
			),
		)

	mux.Handle("/api/users", usersChain)

	// -----------------------------
	// Shops
	// -----------------------------

	shopsCreateChain :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermShopsCreate)(http.HandlerFunc(handler.CreateShops)),
			),
		)

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

	shopsMeChain :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermShopsUpdate)(shopsMeHandler),
			),
		)

	mux.Handle("/api/shops/me", shopsMeChain)

	shopsMeMembersHandler := http.HandlerFunc(handler.ShopsMeMembers)

	shopsMeMembersChain :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermUsersCreate)(shopsMeMembersHandler),
			),
		)

	mux.Handle("/api/shops/me/members", shopsMeMembersChain)

	// -----------------------------
	// Self
	// -----------------------------

	selfHandler := http.HandlerFunc(handler.Self)

	selfChain :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(selfHandler),
		)

	mux.Handle("/api/users/me", selfChain)

	// -----------------------------
	// Dashboard
	// -----------------------------

	dashboardOverview :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermDashboardRead)(http.HandlerFunc(handler.DashboardOverview)),
			),
		)

	mux.Handle("/api/dashboard/overview", dashboardOverview)

	dashboardRevenue :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermDashboardRead)(http.HandlerFunc(handler.DashboardRevenue7d)),
			),
		)

	mux.Handle("/api/dashboard/revenue-7d", dashboardRevenue)

	dashboardLowStock :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermDashboardRead)(http.HandlerFunc(handler.DashboardLowStock)),
			),
		)

	mux.Handle("/api/dashboard/low-stock", dashboardLowStock)

	dashboardKPIs :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermDashboardRead)(http.HandlerFunc(handler.DashboardKPIs)),
			),
		)

	mux.Handle("/api/dashboard/kpis", dashboardKPIs)

	// -----------------------------
	// Affiliate Import
	// -----------------------------

	affiliateImport :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermInventoryCreate)(http.HandlerFunc(handler.AffiliateImport)),
			),
		)

	mux.Handle("/api/affiliate/import", affiliateImport)

	// -----------------------------
	// Analytics
	// -----------------------------

	analyticsRecon :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermAnalyticsRead)(http.HandlerFunc(handler.AnalyticsReconciliation)),
			),
		)

	mux.Handle("/api/analytics/reconciliation", analyticsRecon)

	analyticsDaily :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermAnalyticsRead)(http.HandlerFunc(handler.AnalyticsDailyMetrics)),
			),
		)

	mux.Handle("/api/analytics/daily-metrics", analyticsDaily)

	analyticsProducts :=
		middleware.Auth(jwtCfg)(
			middleware.Tenant(
				middleware.RequirePermission(rbac.PermAnalyticsRead)(http.HandlerFunc(handler.AnalyticsProductMetrics)),
			),
		)

	mux.Handle("/api/analytics/product-metrics", analyticsProducts)

	// -----------------------------
	// HTTP Server
	// -----------------------------

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: middleware.CORS(mux),
	}

	go func() {

		log.Println("server listening on port", port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}

	}()

	// -----------------------------
	// Graceful Shutdown
	// -----------------------------

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("server shutdown error:", err)
	}

	log.Println("server stopped")

}

// ----------------------------------
// DB Connect Retry
// ----------------------------------

func connectDB(cfg database.Config) {

	var err error

	for i := 0; i < 5; i++ {

		_, err = database.Open(cfg)

		if err == nil {
			log.Println("database connected")
			return
		}

		log.Printf("database connect failed (%d/5): %v", i+1, err)

		time.Sleep(time.Duration(i+1) * 2 * time.Second)

	}

	log.Fatalf("database connection failed: %v", err)

}
