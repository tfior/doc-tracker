package cases

import (
	"errors"
	"net/http"

	"github.com/tfior/doc-tracker/platform"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/cases", h.listCases)
	mux.HandleFunc("GET /api/v1/cases/{caseId}", h.getCase)
}

func (h *Handler) listCases(w http.ResponseWriter, r *http.Request) {
	page, perPage, ok := platform.ParsePagination(w, r)
	if !ok {
		return
	}

	items, total, err := h.svc.ListCases(r.Context(), page, perPage)
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusOK, platform.ListResponse{
		Items:   items,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

func (h *Handler) getCase(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")

	detail, err := h.svc.GetCase(r.Context(), caseID)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Case not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusOK, detail)
}
