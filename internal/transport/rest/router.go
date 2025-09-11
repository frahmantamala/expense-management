package rest

import (
	"database/sql"
	"net/http"

	"github.com/frahmantamala/expense-management/internal/auth"
	"github.com/frahmantamala/expense-management/internal/transport/swagger"
	"github.com/go-chi/chi"
)

func RegisterAllRoutes(router *chi.Mux, db *sql.DB, authHandler *auth.Handler) {
	healthHandler := NewHealthHandler(db)

	// Mount API under /api/v1 to match OpenAPI basePath
	router.Route("/api/v1", func(r chi.Router) {
		// OpenAPI spec file
		r.Get("/openapi.yml", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./api/openapi.yml")
		})
		// Swagger UI route
		router.Handle("/swagger/*", swagger.Handler())

		// Health check route
		r.Get("/health", healthHandler.healthCheckHandler)
		r.Get("/ping", healthHandler.pingHandler)

		// Auth routes
		if authHandler != nil {
			r.Route("/auth", func(sr chi.Router) {
				sr.Post("/login", authHandler.Login)
				sr.Post("/refresh", authHandler.RefreshToken)
				sr.Post("/logout", authHandler.Logout)
			})
		}
	})
}
