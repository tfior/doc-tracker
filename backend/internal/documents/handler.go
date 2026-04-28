package documents

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tfior/doc-tracker/platform"
)

var validDocumentTypes = map[string]bool{
	"birth_certificate": true, "marriage_certificate": true,
	"naturalization": true, "death_certificate": true, "other": true,
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/cases/{caseId}/documents", h.listDocuments)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/documents", h.createDocument)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}/documents/{docId}", h.updateDocument)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/documents/{docId}", h.deleteDocument)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}/documents/{docId}/status", h.transitionStatus)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}/documents/{docId}/parent", h.reassignDocument)
}

func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")

	page, perPage, ok := platform.ParsePagination(w, r)
	if !ok {
		return
	}

	items, total, err := h.svc.ListDocuments(r.Context(), caseID, page, perPage)
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

func (h *Handler) createDocument(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")

	var body struct {
		PersonID            string              `json:"person_id"`
		DocumentType        string              `json:"document_type"`
		Title               string              `json:"title"`
		LifeEventID         platform.NullString `json:"life_event_id"`
		IssuingAuthority    platform.NullString `json:"issuing_authority"`
		IssueDate           platform.NullString `json:"issue_date"`
		RecordedDate        platform.NullString `json:"recorded_date"`
		RecordedGivenName   platform.NullString `json:"recorded_given_name"`
		RecordedSurname     platform.NullString `json:"recorded_surname"`
		RecordedBirthDate   platform.NullString `json:"recorded_birth_date"`
		RecordedBirthPlace  platform.NullString `json:"recorded_birth_place"`
		Notes               platform.NullString `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.PersonID == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "person_id is required")
		return
	}
	if body.DocumentType == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "document_type is required")
		return
	}
	if !validDocumentTypes[body.DocumentType] {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid document_type")
		return
	}
	if body.Title == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "title is required")
		return
	}

	input := UpdateDocumentInput{
		LifeEventID:        fromPlatformNull(body.LifeEventID),
		IssuingAuthority:   fromPlatformNull(body.IssuingAuthority),
		IssueDate:          fromPlatformNull(body.IssueDate),
		RecordedDate:       fromPlatformNull(body.RecordedDate),
		RecordedGivenName:  fromPlatformNull(body.RecordedGivenName),
		RecordedSurname:    fromPlatformNull(body.RecordedSurname),
		RecordedBirthDate:  fromPlatformNull(body.RecordedBirthDate),
		RecordedBirthPlace: fromPlatformNull(body.RecordedBirthPlace),
		Notes:              fromPlatformNull(body.Notes),
	}

	d, err := h.svc.CreateDocument(r.Context(), caseID, body.PersonID, body.DocumentType, body.Title, input)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Person or life event not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusCreated, d)
}

func (h *Handler) updateDocument(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	docID := r.PathValue("docId")

	var body struct {
		Title               *string             `json:"title"`
		DocumentType        *string             `json:"document_type"`
		IssuingAuthority    platform.NullString `json:"issuing_authority"`
		IssueDate           platform.NullString `json:"issue_date"`
		RecordedDate        platform.NullString `json:"recorded_date"`
		RecordedGivenName   platform.NullString `json:"recorded_given_name"`
		RecordedSurname     platform.NullString `json:"recorded_surname"`
		RecordedBirthDate   platform.NullString `json:"recorded_birth_date"`
		RecordedBirthPlace  platform.NullString `json:"recorded_birth_place"`
		Notes               platform.NullString `json:"notes"`
		IsVerified          *bool               `json:"is_verified"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.Title != nil && *body.Title == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "title must not be empty")
		return
	}
	if body.DocumentType != nil && !validDocumentTypes[*body.DocumentType] {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid document_type")
		return
	}

	input := UpdateDocumentInput{
		Title:              body.Title,
		DocumentType:       body.DocumentType,
		IssuingAuthority:   fromPlatformNull(body.IssuingAuthority),
		IssueDate:          fromPlatformNull(body.IssueDate),
		RecordedDate:       fromPlatformNull(body.RecordedDate),
		RecordedGivenName:  fromPlatformNull(body.RecordedGivenName),
		RecordedSurname:    fromPlatformNull(body.RecordedSurname),
		RecordedBirthDate:  fromPlatformNull(body.RecordedBirthDate),
		RecordedBirthPlace: fromPlatformNull(body.RecordedBirthPlace),
		Notes:              fromPlatformNull(body.Notes),
		IsVerified:         body.IsVerified,
	}

	d, err := h.svc.UpdateDocument(r.Context(), caseID, docID, input)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Document not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusOK, d)
}

func (h *Handler) deleteDocument(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	docID := r.PathValue("docId")

	err := h.svc.DeleteDocument(r.Context(), caseID, docID)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Document not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) transitionStatus(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	docID := r.PathValue("docId")

	var body struct {
		StatusKey string `json:"status_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if !validStatusKeys[body.StatusKey] {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "status_key must be one of: pending, collected, verified, unobtainable")
		return
	}

	d, err := h.svc.TransitionStatus(r.Context(), caseID, docID, body.StatusKey)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Document not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusOK, d)
}

func (h *Handler) reassignDocument(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	docID := r.PathValue("docId")

	var body struct {
		PersonID    string              `json:"person_id"`
		LifeEventID platform.NullString `json:"life_event_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.PersonID == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "person_id is required")
		return
	}

	input := ReassignDocumentInput{
		PersonID:    body.PersonID,
		LifeEventID: fromPlatformNull(body.LifeEventID),
	}

	d, err := h.svc.ReassignDocument(r.Context(), caseID, docID, input)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Document, person, or life event not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusOK, d)
}

func fromPlatformNull(n platform.NullString) NullableField {
	return NullableField{Set: n.Set, Valid: n.Valid, Value: n.Value}
}
