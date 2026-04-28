package claimlines

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
	mux.HandleFunc("GET /api/v1/cases/{caseId}/claim-lines", h.listClaimLines)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/claim-lines", h.createClaimLine)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}/claim-lines/{lineId}", h.updateClaimLine)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/claim-lines/{lineId}", h.deleteClaimLine)
}

func (h *Handler) listClaimLines(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")

	page, perPage, ok := platform.ParsePagination(w, r)
	if !ok {
		return
	}

	items, total, err := h.svc.ListClaimLines(r.Context(), caseID, page, perPage)
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

func (h *Handler) createClaimLine(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")

	var body struct {
		RootPersonID string              `json:"root_person_id"`
		Status       string              `json:"status"`
		Notes        platform.NullString `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.RootPersonID == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "root_person_id is required")
		return
	}
	status := body.Status
	if status == "" {
		status = "not_yet_researched"
	} else if !validStatuses[status] {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid status")
		return
	}

	cl, err := h.svc.CreateClaimLine(r.Context(), caseID, body.RootPersonID, status, fromPlatformNull(body.Notes))
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Person not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusCreated, cl)
}

func (h *Handler) updateClaimLine(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	lineID := r.PathValue("lineId")

	var body struct {
		Status *string             `json:"status"`
		Notes  platform.NullString `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.Status != nil && !validStatuses[*body.Status] {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid status")
		return
	}

	cl, err := h.svc.UpdateClaimLine(r.Context(), caseID, lineID, body.Status, fromPlatformNull(body.Notes))
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Claim line not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusOK, cl)
}

func (h *Handler) deleteClaimLine(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	lineID := r.PathValue("lineId")

	err := h.svc.DeleteClaimLine(r.Context(), caseID, lineID)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Claim line not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func fromPlatformNull(n platform.NullString) NullableField {
	return NullableField{Set: n.Set, Valid: n.Valid, Value: n.Value}
}
