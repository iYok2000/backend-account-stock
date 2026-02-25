package main

import (
	"log"
	"net/http"
	"os"

	"account-stock-be/internal/auth"
	"account-stock-be/internal/database"
	"account-stock-be/internal/handler"
	"account-stock-be/internal/middleware"
	"account-stock-be/internal/rbac"
)

func main() {
	// Production: refuse to start with dev JWT secret
	if os.Getenv("APP_ENV") == "production" {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" || secret == "dev-secret-change-in-production" {
			log.Fatal("production requires JWT_SECRET to be set and not the dev default")
		}
	}
	jwtCfg := auth.DefaultJWTConfig()

	// Connect to DB when DATABASE_URL or SUPABASE_DB_URL is set (Postgres / Supabase)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("SUPABASE_DB_URL")
	}
	if dsn != "" {
		dbCfg := database.DefaultConfig()
		if _, err := database.Open(dbCfg); err != nil {
			log.Fatalf("database: %v", err)
		}
		defer database.Close()
	}

	mux := http.NewServeMux()

	// Public (no auth)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API: import — stub for frontend Import Wizard (CORS needed for browser)
	mux.HandleFunc("/api/import/order-transaction", handler.ImportOrderTransaction)

	// API: auth — /api/auth/me requires valid JWT
	apiAuth := http.NewServeMux()
	apiAuth.HandleFunc("/me", middleware.RequireAuthContext(handler.Me))
	mux.Handle("/api/auth/", http.StripPrefix("/api/auth", middleware.Auth(jwtCfg)(apiAuth)))

	// API: users — Auth then RequirePermission(users:read) per RBAC_SPEC
	usersHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			middleware.WriteJSONError(w, middleware.ErrMethodNotAllowed, http.StatusMethodNotAllowed)
			return
		}
		handler.UsersList(w, r)
	})
	usersChain := middleware.Auth(jwtCfg)(middleware.RequirePermission(rbac.PermUsersRead)(usersHandler))
	mux.Handle("/api/users", usersChain)

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
