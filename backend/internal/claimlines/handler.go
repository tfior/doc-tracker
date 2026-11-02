// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package claimlines

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
	mux.HandleFunc("GET /api/v1/cases/{caseId}/claim-lines", h.listClaimLines)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/claim-lines", h.createClaimLine)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}/claim-lines/{lineId}", h.updateClaimLine)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/claim-lines/{lineId}", h.deleteClaimLine)
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

func (h *Handler) listClaimLines(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListClaimLines(r.Context(), r.PathValue("caseId"))
	if err != nil {
		respondError(w, r, err, "Case not found")
		return
	}
	platform.JSON(w, http.StatusOK, platform.Items{Items: items, Total: len(items)})
}

func (h *Handler) createClaimLine(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RootPersonID string              `json:"root_person_id"`
		Status       string              `json:"status"`
		Notes        platform.NullString `json:"notes"`
	}
	if err := platform.Decode(r, &body); err != nil {
		respondError(w, r, err, "")
		return
	}
	var v platform.Validator
	v.Required("root_person_id", body.RootPersonID)
	v.ID("root_person_id", body.RootPersonID)
	v.OneOf("status", body.Status, validStatuses)
	if err := v.Err(); err != nil {
		respondError(w, r, err, "")
		return
	}
	status := body.Status
	if status == "" {
		status = "not_yet_researched"
	}
	cl, err := h.svc.CreateClaimLine(r.Context(), r.PathValue("caseId"), body.RootPersonID, status, body.Notes)
	if err != nil {
		respondError(w, r, err, "Person not found")
		return
	}
	platform.JSON(w, http.StatusCreated, cl)
}

func (h *Handler) updateClaimLine(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Version *int                `json:"version"`
		Status  *string             `json:"status"`
		Notes   platform.NullString `json:"notes"`
	}
	if err := platform.Decode(r, &body); err != nil {
		respondError(w, r, err, "")
		return
	}
	var v platform.Validator
	v.Version(body.Version)
	if body.Status != nil {
		v.OneOf("status", *body.Status, validStatuses)
	}
	if err := v.Err(); err != nil {
		respondError(w, r, err, "")
		return
	}
	cl, err := h.svc.UpdateClaimLine(r.Context(), r.PathValue("caseId"), r.PathValue("lineId"), *body.Version, body.Status, body.Notes)
	if err != nil {
		respondError(w, r, err, "Claim line not found")
		return
	}
	platform.JSON(w, http.StatusOK, cl)
}

func (h *Handler) deleteClaimLine(w http.ResponseWriter, r *http.Request) {
	version, err := platform.QueryVersion(r)
	if err != nil {
		respondError(w, r, err, "")
		return
	}
	if err := h.svc.DeleteClaimLine(r.Context(), r.PathValue("caseId"), r.PathValue("lineId"), version); err != nil {
		respondError(w, r, err, "Claim line not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
