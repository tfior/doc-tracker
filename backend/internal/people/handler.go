// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package people

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
	mux.HandleFunc("GET /api/v1/cases/{caseId}/people", h.listPeople)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/people", h.createPerson)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}/people/{personId}", h.updatePerson)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/people/{personId}", h.deletePerson)
	mux.HandleFunc("GET /api/v1/cases/{caseId}/people/{personId}/deletion-impact", h.deletionImpact)
}

// respondError writes the response for an error from the service.
func respondError(w http.ResponseWriter, r *http.Request, err error, notFound string) {
	if platform.WriteError(w, err) {
		return
	}
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", notFound)
		return
	}
	platform.ServerError(w, r, err)
}

// personFields is the body shared by create and update.
type personFields struct {
	FirstName  *string                    `json:"first_name"`
	LastName   *string                    `json:"last_name"`
	BirthDate  platform.NullQualifiedDate `json:"birth_date"`
	BirthPlace platform.NullString        `json:"birth_place"`
	DeathDate  platform.NullQualifiedDate `json:"death_date"`
	DeathPlace platform.NullString        `json:"death_place"`
	Notes      platform.NullString        `json:"notes"`
	ParentIDs  *[]string                  `json:"parent_ids"`
	ChildIDs   *[]string                  `json:"child_ids"`
}

func (f *personFields) validate(v *platform.Validator) {
	platform.Trim(f.FirstName)
	platform.Trim(f.LastName)
	v.NotBlank("first_name", f.FirstName)
	v.NotBlank("last_name", f.LastName)
	v.QualifiedDate("birth_date", &f.BirthDate)
	v.QualifiedDate("death_date", &f.DeathDate)
	if f.ParentIDs != nil {
		v.IDs("parent_ids", *f.ParentIDs)
	}
	if f.ChildIDs != nil {
		v.IDs("child_ids", *f.ChildIDs)
	}
}

func (f *personFields) input() UpdatePersonInput {
	return UpdatePersonInput{
		FirstName: f.FirstName, LastName: f.LastName,
		BirthDate: f.BirthDate, BirthPlace: f.BirthPlace, DeathDate: f.DeathDate, DeathPlace: f.DeathPlace,
		Notes:     f.Notes,
		ParentIDs: f.ParentIDs, ChildIDs: f.ChildIDs,
	}
}

func (h *Handler) listPeople(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListPeople(r.Context(), r.PathValue("caseId"))
	if err != nil {
		respondError(w, r, err, "Case not found")
		return
	}
	platform.JSON(w, http.StatusOK, platform.Items{Items: items, Total: len(items)})
}

func (h *Handler) createPerson(w http.ResponseWriter, r *http.Request) {
	var body personFields
	if err := platform.Decode(r, &body); err != nil {
		respondError(w, r, err, "")
		return
	}
	var v platform.Validator
	if body.FirstName == nil {
		v.Required("first_name", "")
	}
	if body.LastName == nil {
		v.Required("last_name", "")
	}
	body.validate(&v)
	if err := v.Err(); err != nil {
		respondError(w, r, err, "")
		return
	}
	p, err := h.svc.CreatePerson(r.Context(), r.PathValue("caseId"), *body.FirstName, *body.LastName, body.input())
	if err != nil {
		respondError(w, r, err, "Case or person not found")
		return
	}
	platform.JSON(w, http.StatusCreated, p)
}

func (h *Handler) updatePerson(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Version *int `json:"version"`
		personFields
	}
	if err := platform.Decode(r, &body); err != nil {
		respondError(w, r, err, "")
		return
	}
	var v platform.Validator
	v.Version(body.Version)
	body.validate(&v)
	if err := v.Err(); err != nil {
		respondError(w, r, err, "")
		return
	}
	input := body.input()
	input.Version = *body.Version
	p, err := h.svc.UpdatePerson(r.Context(), r.PathValue("caseId"), r.PathValue("personId"), input)
	if err != nil {
		respondError(w, r, err, "Person not found")
		return
	}
	platform.JSON(w, http.StatusOK, p)
}

func (h *Handler) deletePerson(w http.ResponseWriter, r *http.Request) {
	version, err := platform.QueryVersion(r)
	if err != nil {
		respondError(w, r, err, "")
		return
	}
	if err := h.svc.DeletePerson(r.Context(), r.PathValue("caseId"), r.PathValue("personId"), version); err != nil {
		respondError(w, r, err, "Person not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deletionImpact(w http.ResponseWriter, r *http.Request) {
	impact, err := h.svc.DeletionImpact(r.Context(), r.PathValue("caseId"), r.PathValue("personId"))
	if err != nil {
		respondError(w, r, err, "Person not found")
		return
	}
	platform.JSON(w, http.StatusOK, impact)
}
