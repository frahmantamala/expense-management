package rest

import (
	"database/sql"
	"net/http"

	"github.com/frahmantamala/expense-management/internal/auth"
	"github.com/frahmantamala/expense-management/internal/transport/swagger"
	"github.com/frahmantamala/expense-management/internal/user"
	"github.com/go-chi/chi"
)

func RegisterAllRoutes(router *chi.Mux, db *sql.DB, authHandler *auth.Handler, userHandler *user.Handler) {
	healthHandler := NewHealthHandler(db)
	// Serve OpenAPI spec at root (outside API prefix)
	router.Get("/openapi.yml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./api/openapi.yml")
	})
	// Swagger UI route at root
	router.Handle("/swagger/*", swagger.Handler())

	// Mount API under /api/v1 to match OpenAPI basePath
	router.Route("/api/v1", func(r chi.Router) {
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
			// Current user
			if userHandler != nil {
				r.With(authHandler.AuthMiddleware).Get("/users/me", userHandler.GetCurrentUser)
			}
		}
	})
}
