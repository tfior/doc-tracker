package trash

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
	mux.HandleFunc("GET /api/v1/trash", h.getGlobalTrash)
	mux.HandleFunc("GET /api/v1/cases/{caseId}/trash", h.getCaseTrash)

	mux.HandleFunc("POST /api/v1/cases/{caseId}/restore", h.restoreCase)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/permanent", h.permanentDeleteCase)

	mux.HandleFunc("POST /api/v1/cases/{caseId}/people/{personId}/restore", h.restorePerson)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/people/{personId}/permanent", h.permanentDeletePerson)

	mux.HandleFunc("POST /api/v1/cases/{caseId}/life-events/{eventId}/restore", h.restoreLifeEvent)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/life-events/{eventId}/permanent", h.permanentDeleteLifeEvent)

	mux.HandleFunc("POST /api/v1/cases/{caseId}/documents/{docId}/restore", h.restoreDocument)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/documents/{docId}/permanent", h.permanentDeleteDocument)

	mux.HandleFunc("POST /api/v1/cases/{caseId}/claim-lines/{lineId}/restore", h.restoreClaimLine)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/claim-lines/{lineId}/permanent", h.permanentDeleteClaimLine)
}

func (h *Handler) getGlobalTrash(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.GetGlobalTrash(r.Context())
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}
	platform.JSON(w, http.StatusOK, t)
}

func (h *Handler) getCaseTrash(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	t, err := h.svc.GetCaseTrash(r.Context(), caseID)
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}
	platform.JSON(w, http.StatusOK, t)
}

func (h *Handler) restoreCase(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	h.handleRestore(w, r, func() error { return h.svc.RestoreCase(r.Context(), caseID) })
}

func (h *Handler) permanentDeleteCase(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	h.handlePermanentDelete(w, r, func() error { return h.svc.PermanentDeleteCase(r.Context(), caseID) })
}

func (h *Handler) restorePerson(w http.ResponseWriter, r *http.Request) {
	caseID, personID := r.PathValue("caseId"), r.PathValue("personId")
	h.handleRestore(w, r, func() error { return h.svc.RestorePerson(r.Context(), caseID, personID) })
}

func (h *Handler) permanentDeletePerson(w http.ResponseWriter, r *http.Request) {
	caseID, personID := r.PathValue("caseId"), r.PathValue("personId")
	h.handlePermanentDelete(w, r, func() error { return h.svc.PermanentDeletePerson(r.Context(), caseID, personID) })
}

func (h *Handler) restoreLifeEvent(w http.ResponseWriter, r *http.Request) {
	caseID, eventID := r.PathValue("caseId"), r.PathValue("eventId")
	h.handleRestore(w, r, func() error { return h.svc.RestoreLifeEvent(r.Context(), caseID, eventID) })
}

func (h *Handler) permanentDeleteLifeEvent(w http.ResponseWriter, r *http.Request) {
	caseID, eventID := r.PathValue("caseId"), r.PathValue("eventId")
	h.handlePermanentDelete(w, r, func() error { return h.svc.PermanentDeleteLifeEvent(r.Context(), caseID, eventID) })
}

func (h *Handler) restoreDocument(w http.ResponseWriter, r *http.Request) {
	caseID, docID := r.PathValue("caseId"), r.PathValue("docId")
	h.handleRestore(w, r, func() error { return h.svc.RestoreDocument(r.Context(), caseID, docID) })
}

func (h *Handler) permanentDeleteDocument(w http.ResponseWriter, r *http.Request) {
	caseID, docID := r.PathValue("caseId"), r.PathValue("docId")
	h.handlePermanentDelete(w, r, func() error { return h.svc.PermanentDeleteDocument(r.Context(), caseID, docID) })
}

func (h *Handler) restoreClaimLine(w http.ResponseWriter, r *http.Request) {
	caseID, lineID := r.PathValue("caseId"), r.PathValue("lineId")
	h.handleRestore(w, r, func() error { return h.svc.RestoreClaimLine(r.Context(), caseID, lineID) })
}

func (h *Handler) permanentDeleteClaimLine(w http.ResponseWriter, r *http.Request) {
	caseID, lineID := r.PathValue("caseId"), r.PathValue("lineId")
	h.handlePermanentDelete(w, r, func() error { return h.svc.PermanentDeleteClaimLine(r.Context(), caseID, lineID) })
}

func (h *Handler) handleRestore(w http.ResponseWriter, _ *http.Request, fn func() error) {
	if err := fn(); err != nil {
		if errors.Is(err, ErrNotFound) {
			platform.Error(w, http.StatusNotFound, "not_found", "Entity not found in trash")
			return
		}
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handlePermanentDelete(w http.ResponseWriter, _ *http.Request, fn func() error) {
	if err := fn(); err != nil {
		if errors.Is(err, ErrNotFound) {
			platform.Error(w, http.StatusNotFound, "not_found", "Entity not found in trash")
			return
		}
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
