package category

import (
	"net/http"

	"github.com/frahmantamala/expense-management/internal/transport"
)

type Handler struct {
	*transport.BaseHandler
	Service *Service
}

func NewHandler(baseHandler *transport.BaseHandler, service *Service) *Handler {
	return &Handler{
		BaseHandler: baseHandler,
		Service:     service,
	}
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.Service.GetAllCategories()
	if err != nil {
		h.Logger.Error("GetCategories: failed to get categories", "error", err)
		h.WriteError(w, http.StatusInternalServerError, "failed to get categories")
		return
	}

	h.WriteJSON(w, http.StatusOK, CategoriesResponse{
		Categories: categories,
	})
}
