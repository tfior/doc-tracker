package cases

import (
	"encoding/json"
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
	mux.HandleFunc("POST /api/v1/cases", h.createCase)
	mux.HandleFunc("GET /api/v1/cases/{caseId}", h.getCase)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}", h.updateCase)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}", h.deleteCase)
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

func (h *Handler) createCase(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.Title == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "title is required")
		return
	}

	c, err := h.svc.CreateCase(r.Context(), body.Title)
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusCreated, c)
}

func (h *Handler) updateCase(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")

	var body struct {
		Title  *string `json:"title"`
		Status *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.Title != nil && *body.Title == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "title must not be empty")
		return
	}
	if body.Status != nil {
		switch *body.Status {
		case "active", "archived", "complete":
		default:
			platform.Error(w, http.StatusBadRequest, "invalid_input", "status must be active, archived, or complete")
			return
		}
	}

	c, err := h.svc.UpdateCase(r.Context(), caseID, body.Title, body.Status)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Case not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusOK, c)
}

func (h *Handler) deleteCase(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")

	err := h.svc.DeleteCase(r.Context(), caseID)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Case not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
