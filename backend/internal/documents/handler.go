// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package documents

import (
	"errors"
	"net/http"
	"time"

	"github.com/tfior/doc-tracker/platform"
)

var validDocumentTypes = map[string]bool{
	"birth_certificate": true, "marriage_certificate": true,
	"naturalization": true, "death_certificate": true,
	"non_naturalization_certificate": true, "divorce_record": true, "court_order": true,
	"other": true,
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/document-statuses", h.listDocumentStatuses)
	mux.HandleFunc("GET /api/v1/cases/{caseId}/documents", h.listDocuments)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/documents", h.createDocument)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}/documents/{docId}", h.updateDocument)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/documents/{docId}", h.deleteDocument)
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

// documentFields is the body shared by create and update.
type documentFields struct {
	LifeEventID        platform.NullString `json:"life_event_id"`
	DocumentType       *string             `json:"document_type"`
	Title              *string             `json:"title"`
	IssuingAuthority   platform.NullString `json:"issuing_authority"`
	IssueDate          platform.NullString `json:"issue_date"`
	RegistrationDate   platform.NullString `json:"registration_date"`
	RecordedGivenName  platform.NullString `json:"recorded_given_name"`
	RecordedSurname    platform.NullString `json:"recorded_surname"`
	RecordedBirthDate  platform.NullString `json:"recorded_birth_date"`
	RecordedBirthPlace platform.NullString `json:"recorded_birth_place"`
	Notes              platform.NullString `json:"notes"`
	DetailsChecked     *bool               `json:"details_checked"`
	ForSubmission      *bool               `json:"for_submission"`
}

func (f *documentFields) validate(v *platform.Validator) {
	platform.Trim(f.Title)
	v.NotBlank("title", f.Title)
	if f.DocumentType != nil {
		v.OneOf("document_type", *f.DocumentType, validDocumentTypes)
	}
	if f.LifeEventID.Set && f.LifeEventID.Valid {
		v.ID("life_event_id", f.LifeEventID.Value)
	}
	v.PartialDate("issue_date", f.IssueDate)
	v.PartialDate("registration_date", f.RegistrationDate)
	v.PartialDate("recorded_birth_date", f.RecordedBirthDate)
}

func (f *documentFields) input() UpdateDocumentInput {
	return UpdateDocumentInput{
		LifeEventID: f.LifeEventID, DocumentType: f.DocumentType, Title: f.Title,
		IssuingAuthority: f.IssuingAuthority, IssueDate: f.IssueDate, RegistrationDate: f.RegistrationDate,
		RecordedGivenName: f.RecordedGivenName, RecordedSurname: f.RecordedSurname,
		RecordedBirthDate: f.RecordedBirthDate, RecordedBirthPlace: f.RecordedBirthPlace,
		Notes: f.Notes, DetailsChecked: f.DetailsChecked, ForSubmission: f.ForSubmission,
	}
}

func (h *Handler) listDocumentStatuses(w http.ResponseWriter, r *http.Request) {
	phase := r.URL.Query().Get("phase")
	if phase != "" && !validPhases[phase] {
		respondError(w, r, platform.InvalidField("phase", "invalid_value",
			"phase must be one of: official_copy, amendment, apostille, translation"), "")
		return
	}
	statuses, err := h.svc.ListDocumentStatuses(r.Context(), phase)
	if err != nil {
		platform.ServerError(w, r, err)
		return
	}
	platform.JSON(w, http.StatusOK, statuses)
}

func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListDocuments(r.Context(), r.PathValue("caseId"))
	if err != nil {
		respondError(w, r, err, "Case not found")
		return
	}
	platform.JSON(w, http.StatusOK, platform.Items{Items: items, Total: len(items)})
}

func (h *Handler) createDocument(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PersonID string `json:"person_id"`
		documentFields
	}
	if err := platform.Decode(r, &body); err != nil {
		respondError(w, r, err, "")
		return
	}
	var v platform.Validator
	v.Required("person_id", body.PersonID)
	v.ID("person_id", body.PersonID)
	if body.DocumentType == nil {
		v.Required("document_type", "")
	}
	if body.Title == nil {
		v.Required("title", "")
	}
	body.validate(&v)
	if err := v.Err(); err != nil {
		respondError(w, r, err, "")
		return
	}
	d, err := h.svc.CreateDocument(r.Context(), r.PathValue("caseId"), body.PersonID, *body.DocumentType, *body.Title, body.input())
	if err != nil {
		respondError(w, r, err, "Person or life event not found")
		return
	}
	platform.JSON(w, http.StatusCreated, d)
}

// phaseFields is one phase's part of an update body.
type phaseFields struct {
	statusID  *string
	since     platform.NullString
	reference platform.NullString
}

// validate checks the phase's fields, named with prefix, and returns them as
// input. A since may not be null or in the future.
func (p *phaseFields) validate(v *platform.Validator, prefix string, now time.Time) PhaseInput {
	in := PhaseInput{StatusID: p.statusID, Reference: p.reference}
	if p.statusID != nil {
		v.Required(prefix+"_status_id", *p.statusID)
		v.ID(prefix+"_status_id", *p.statusID)
	}
	field := prefix + "_status_since"
	switch t, ok := platform.ParseTimestamp(p.since.Value); {
	case !p.since.Set:
	case !p.since.Valid:
		v.Add(field, "required", "This field cannot be empty.")
	case !ok:
		v.Add(field, "invalid_timestamp", "Enter a date and time, for example 2026-03-15T00:00:00Z.")
	case t.After(now):
		v.Add(field, "in_future", "This date cannot be in the future.")
	default:
		in.Since = &t
	}
	return in
}

func (h *Handler) updateDocument(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Version                 *int                `json:"version"`
		PersonID                *string             `json:"person_id"`
		OfficialCopyStatusID    *string             `json:"official_copy_status_id"`
		OfficialCopyStatusSince platform.NullString `json:"official_copy_status_since"`
		OfficialCopyReference   platform.NullString `json:"official_copy_reference"`
		AmendmentStatusID       *string             `json:"amendment_status_id"`
		AmendmentStatusSince    platform.NullString `json:"amendment_status_since"`
		AmendmentReference      platform.NullString `json:"amendment_reference"`
		ApostilleStatusID       *string             `json:"apostille_status_id"`
		ApostilleStatusSince    platform.NullString `json:"apostille_status_since"`
		ApostilleReference      platform.NullString `json:"apostille_reference"`
		TranslationStatusID     *string             `json:"translation_status_id"`
		TranslationStatusSince  platform.NullString `json:"translation_status_since"`
		TranslationReference    platform.NullString `json:"translation_reference"`
		documentFields
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
	now := time.Now()
	officialCopy := (&phaseFields{body.OfficialCopyStatusID, body.OfficialCopyStatusSince, body.OfficialCopyReference}).validate(&v, "official_copy", now)
	amendment := (&phaseFields{body.AmendmentStatusID, body.AmendmentStatusSince, body.AmendmentReference}).validate(&v, "amendment", now)
	apostille := (&phaseFields{body.ApostilleStatusID, body.ApostilleStatusSince, body.ApostilleReference}).validate(&v, "apostille", now)
	translation := (&phaseFields{body.TranslationStatusID, body.TranslationStatusSince, body.TranslationReference}).validate(&v, "translation", now)
	body.validate(&v)
	if err := v.Err(); err != nil {
		respondError(w, r, err, "")
		return
	}
	input := body.input()
	input.Version = *body.Version
	input.PersonID = body.PersonID
	input.OfficialCopy, input.Amendment, input.Apostille, input.Translation = officialCopy, amendment, apostille, translation
	d, err := h.svc.UpdateDocument(r.Context(), r.PathValue("caseId"), r.PathValue("docId"), input)
	if err != nil {
		respondError(w, r, err, "Document, person, or life event not found")
		return
	}
	platform.JSON(w, http.StatusOK, d)
}

func (h *Handler) deleteDocument(w http.ResponseWriter, r *http.Request) {
	version, err := platform.QueryVersion(r)
	if err != nil {
		respondError(w, r, err, "")
		return
	}
	if err := h.svc.DeleteDocument(r.Context(), r.PathValue("caseId"), r.PathValue("docId"), version); err != nil {
		respondError(w, r, err, "Document not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
