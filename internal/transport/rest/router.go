package rest

import (
	"database/sql"
	"net/http"

	"github.com/frahmantamala/expense-management/internal/auth"
	"github.com/frahmantamala/expense-management/internal/category"
	"github.com/frahmantamala/expense-management/internal/expense"
	"github.com/frahmantamala/expense-management/internal/payment"
	"github.com/frahmantamala/expense-management/internal/transport/middleware"
	"github.com/frahmantamala/expense-management/internal/transport/swagger"
	"github.com/frahmantamala/expense-management/internal/user"
	"github.com/go-chi/chi"
)

func RegisterAllRoutes(router *chi.Mux, db *sql.DB, authHandler *auth.Handler, userHandler *user.Handler, expenseHandler *expense.Handler, categoryHandler *category.Handler, paymentHandler *payment.Handler) {
	healthHandler := NewHealthHandler(db)

	// Apply CORS middleware to all routes
	router.Use(middleware.CORS)

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
		}

		// Public categories route (no auth required)
		if categoryHandler != nil {
			r.Get("/categories", categoryHandler.GetCategories)
		}

		if authHandler != nil {
			// Protected routes that require authentication
			r.Group(func(pr chi.Router) {
				pr.Use(authHandler.AuthMiddleware)

				// Current user
				if userHandler != nil {
					pr.Get("/users/me", userHandler.GetCurrentUser)
				}

				// Expense routes
				if expenseHandler != nil {
					pr.Route("/expenses", func(er chi.Router) {
						// User expense routes
						er.Post("/", expenseHandler.CreateExpense) // POST /expenses
						er.Get("/", expenseHandler.GetAllExpenses) // GET /expenses
						er.Get("/{id}", expenseHandler.GetExpense) // GET /expenses/:id

						// Manager routes with permission protection
						er.Group(func(mr chi.Router) {
							mr.Use(middleware.RequirePermissions("approve_expenses", "reject_expenses", "manager", "admin"))
							mr.Patch("/{id}/approve", expenseHandler.ApproveExpense) // PATCH /expenses/:id/approve
							mr.Patch("/{id}/reject", expenseHandler.RejectExpense)   // PATCH /expenses/:id/reject
						})
					})
				}

				// Payment routes (requires retry_payments permission)
				if paymentHandler != nil {
					pr.Group(func(pmr chi.Router) {
						pmr.Use(middleware.RequirePermissions("retry_payments", "admin"))
						pmr.Post("/payment/retry", paymentHandler.RetryPayment) // POST /payment/retry
					})
				}
			})
		}
	})
}
