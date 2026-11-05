// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package lifeevents

import (
	"errors"
	"net/http"

	"github.com/tfior/doc-tracker/platform"
)

var validEventTypes = map[string]bool{
	"birth": true, "marriage": true, "death": true,
	"naturalization": true, "immigration": true, "other": true,
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/cases/{caseId}/life-events", h.listLifeEvents)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/life-events", h.createLifeEvent)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}/life-events/{eventId}", h.updateLifeEvent)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/life-events/{eventId}", h.deleteLifeEvent)
	mux.HandleFunc("GET /api/v1/cases/{caseId}/life-events/{eventId}/deletion-impact", h.deletionImpact)
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

// eventFields is the body shared by create and update.
type eventFields struct {
	EventType        *string                    `json:"event_type"`
	EventDate        platform.NullQualifiedDate `json:"event_date"`
	EventPlace       platform.NullString        `json:"event_place"`
	SpousePersonID   platform.NullString        `json:"spouse_person_id"`
	SpouseName       platform.NullString        `json:"spouse_name"`
	SpouseBirthDate  platform.NullQualifiedDate `json:"spouse_birth_date"`
	SpouseBirthPlace platform.NullString        `json:"spouse_birth_place"`
	DivorceDate      platform.NullQualifiedDate `json:"divorce_date"`
	DivorcePlace     platform.NullString        `json:"divorce_place"`
	Notes            platform.NullString        `json:"notes"`
}

func (f *eventFields) validate(v *platform.Validator) {
	if f.EventType != nil {
		v.OneOf("event_type", *f.EventType, validEventTypes)
	}
	v.QualifiedDate("event_date", &f.EventDate)
	v.ID("spouse_person_id", f.SpousePersonID.Value)
	v.QualifiedDate("spouse_birth_date", &f.SpouseBirthDate)
	v.QualifiedDate("divorce_date", &f.DivorceDate)
}

func (f *eventFields) input() UpdateLifeEventInput {
	return UpdateLifeEventInput{
		EventType: f.EventType, EventDate: f.EventDate, EventPlace: f.EventPlace,
		SpousePersonID: f.SpousePersonID,
		SpouseName:     f.SpouseName, SpouseBirthDate: f.SpouseBirthDate, SpouseBirthPlace: f.SpouseBirthPlace,
		DivorceDate: f.DivorceDate, DivorcePlace: f.DivorcePlace,
		Notes: f.Notes,
	}
}

func (h *Handler) listLifeEvents(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListLifeEvents(r.Context(), r.PathValue("caseId"))
	if err != nil {
		respondError(w, r, err, "Case not found")
		return
	}
	platform.JSON(w, http.StatusOK, platform.Items{Items: items, Total: len(items)})
}

func (h *Handler) createLifeEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PersonID string `json:"person_id"`
		eventFields
	}
	if err := platform.Decode(r, &body); err != nil {
		respondError(w, r, err, "")
		return
	}
	var v platform.Validator
	v.Required("person_id", body.PersonID)
	v.ID("person_id", body.PersonID)
	if body.EventType == nil {
		v.Required("event_type", "")
	}
	body.validate(&v)
	if err := v.Err(); err != nil {
		respondError(w, r, err, "")
		return
	}
	le, err := h.svc.CreateLifeEvent(r.Context(), r.PathValue("caseId"), body.PersonID, *body.EventType, body.input())
	if err != nil {
		respondError(w, r, err, "Person not found")
		return
	}
	platform.JSON(w, http.StatusCreated, le)
}

func (h *Handler) updateLifeEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Version  *int    `json:"version"`
		PersonID *string `json:"person_id"`
		eventFields
	}
	if err := platform.Decode(r, &body); err != nil {
		respondError(w, r, err, "")
		return
	}
	var v platform.Validator
	v.Version(body.Version)
	if body.PersonID != nil {
		v.Required("person_id", *body.PersonID)
		v.ID("person_id", *body.PersonID)
	}
	body.validate(&v)
	if err := v.Err(); err != nil {
		respondError(w, r, err, "")
		return
	}
	input := body.input()
	input.Version = *body.Version
	input.PersonID = body.PersonID
	le, err := h.svc.UpdateLifeEvent(r.Context(), r.PathValue("caseId"), r.PathValue("eventId"), input)
	if err != nil {
		respondError(w, r, err, "Life event or person not found")
		return
	}
	platform.JSON(w, http.StatusOK, le)
}

func (h *Handler) deleteLifeEvent(w http.ResponseWriter, r *http.Request) {
	version, err := platform.QueryVersion(r)
	if err != nil {
		respondError(w, r, err, "")
		return
	}
	if err := h.svc.DeleteLifeEvent(r.Context(), r.PathValue("caseId"), r.PathValue("eventId"), version); err != nil {
		respondError(w, r, err, "Life event not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deletionImpact(w http.ResponseWriter, r *http.Request) {
	impact, err := h.svc.DeletionImpact(r.Context(), r.PathValue("caseId"), r.PathValue("eventId"))
	if err != nil {
		respondError(w, r, err, "Life event not found")
		return
	}
	platform.JSON(w, http.StatusOK, impact)
}
