package category

import (
	"net/http"

	"github.com/frahmantamala/expense-management/internal/transport"
)

type ServiceAPI interface {
	GetAllCategories() ([]CategoryResponse, error)
	GetCategoryByName(name string) (*CategoryResponse, error)
	IsValidCategory(name string) bool
}

type Handler struct {
	*transport.BaseHandler
	Service ServiceAPI
}

func NewHandler(baseHandler *transport.BaseHandler, service ServiceAPI) *Handler {
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
