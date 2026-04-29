package claimlines

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tfior/doc-tracker/internal/activitylog"
	"github.com/tfior/doc-tracker/internal/auth"
	"github.com/tfior/doc-tracker/platform"
)

type Handler struct {
	svc    *Service
	actlog *activitylog.Service
}

func NewHandler(svc *Service, actlog *activitylog.Service) *Handler {
	return &Handler{svc: svc, actlog: actlog}
}

func (h *Handler) log(r *http.Request, p activitylog.InsertParams) {
	if h.actlog == nil {
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	p.UserID = userID
	_ = h.actlog.Insert(r.Context(), p)
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

	h.log(r, activitylog.InsertParams{
		CaseID: caseID, Action: "created", EntityType: "claim_line",
		EntityID: cl.ID, EntityName: cl.Status + " claim line",
	})
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

	var changes []activitylog.FieldChange
	if body.Status != nil {
		changes = append(changes, activitylog.FieldChange{Field: "status", To: *body.Status})
	}
	if body.Notes.Set {
		var val interface{}
		if body.Notes.Valid {
			val = body.Notes.Value
		}
		changes = append(changes, activitylog.FieldChange{Field: "notes", To: val})
	}
	h.log(r, activitylog.InsertParams{
		CaseID: caseID, Action: "updated", EntityType: "claim_line",
		EntityID: cl.ID, EntityName: cl.Status + " claim line", Changes: changes,
	})
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

	h.log(r, activitylog.InsertParams{
		CaseID: caseID, Action: "deleted", EntityType: "claim_line",
		EntityID: lineID, EntityName: lineID,
	})
	w.WriteHeader(http.StatusNoContent)
}

func fromPlatformNull(n platform.NullString) NullableField {
	return NullableField{Set: n.Set, Valid: n.Valid, Value: n.Value}
}
