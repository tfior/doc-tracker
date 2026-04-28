package people

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
	mux.HandleFunc("GET /api/v1/cases/{caseId}/people", h.listPeople)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/people", h.createPerson)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}/people/{personId}", h.updatePerson)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/people/{personId}", h.deletePerson)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/people/{personId}/relationships", h.addParent)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/people/{personId}/relationships/{parentId}", h.removeParent)
}

func (h *Handler) listPeople(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")

	page, perPage, ok := platform.ParsePagination(w, r)
	if !ok {
		return
	}

	items, total, err := h.svc.ListPeople(r.Context(), caseID, page, perPage)
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

func (h *Handler) createPerson(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")

	var body struct {
		FirstName  string               `json:"first_name"`
		LastName   string               `json:"last_name"`
		BirthDate  platform.NullString  `json:"birth_date"`
		BirthPlace platform.NullString  `json:"birth_place"`
		DeathDate  platform.NullString  `json:"death_date"`
		Notes      platform.NullString  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.FirstName == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "first_name is required")
		return
	}
	if body.LastName == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "last_name is required")
		return
	}

	input := UpdatePersonInput{
		BirthDate:  fromPlatformNull(body.BirthDate),
		BirthPlace: fromPlatformNull(body.BirthPlace),
		DeathDate:  fromPlatformNull(body.DeathDate),
		Notes:      fromPlatformNull(body.Notes),
	}

	p, err := h.svc.CreatePerson(r.Context(), caseID, body.FirstName, body.LastName, input)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Case not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusCreated, p)
}

func (h *Handler) updatePerson(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	personID := r.PathValue("personId")

	var body struct {
		FirstName  *string              `json:"first_name"`
		LastName   *string              `json:"last_name"`
		BirthDate  platform.NullString  `json:"birth_date"`
		BirthPlace platform.NullString  `json:"birth_place"`
		DeathDate  platform.NullString  `json:"death_date"`
		Notes      platform.NullString  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.FirstName != nil && *body.FirstName == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "first_name must not be empty")
		return
	}
	if body.LastName != nil && *body.LastName == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "last_name must not be empty")
		return
	}

	input := UpdatePersonInput{
		FirstName:  body.FirstName,
		LastName:   body.LastName,
		BirthDate:  fromPlatformNull(body.BirthDate),
		BirthPlace: fromPlatformNull(body.BirthPlace),
		DeathDate:  fromPlatformNull(body.DeathDate),
		Notes:      fromPlatformNull(body.Notes),
	}

	p, err := h.svc.UpdatePerson(r.Context(), caseID, personID, input)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Person not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusOK, p)
}

func (h *Handler) deletePerson(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	personID := r.PathValue("personId")

	err := h.svc.DeletePerson(r.Context(), caseID, personID)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Person not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) addParent(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	personID := r.PathValue("personId")

	var body struct {
		ParentID string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.ParentID == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "parent_id is required")
		return
	}
	if body.ParentID == personID {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "a person cannot be their own parent")
		return
	}

	rel, err := h.svc.AddParent(r.Context(), caseID, personID, body.ParentID)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Person not found")
		return
	}
	if errors.Is(err, ErrConflict) {
		platform.Error(w, http.StatusConflict, "conflict", "Relationship already exists or parent limit reached")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	platform.JSON(w, http.StatusCreated, rel)
}

func (h *Handler) removeParent(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	personID := r.PathValue("personId")
	parentID := r.PathValue("parentId")

	err := h.svc.RemoveParent(r.Context(), caseID, personID, parentID)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Relationship not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// fromPlatformNull converts a platform.NullString to the internal NullableField type.
func fromPlatformNull(n platform.NullString) NullableField {
	return NullableField{Set: n.Set, Valid: n.Valid, Value: n.Value}
}
