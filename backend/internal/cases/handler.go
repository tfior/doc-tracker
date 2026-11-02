// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cases

import (
	"errors"
	"net/http"

	"github.com/tfior/doc-tracker/platform"
)

var validStatuses = map[string]bool{"active": true, "archived": true, "complete": true}

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

// respondError writes the response for an error from the service.
func respondError(w http.ResponseWriter, r *http.Request, err error) {
	if platform.WriteError(w, err) {
		return
	}
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Case not found")
		return
	}
	platform.ServerError(w, r, err)
}

func (h *Handler) listCases(w http.ResponseWriter, r *http.Request) {
	page, perPage, ok := platform.ParsePagination(w, r)
	if !ok {
		return
	}
	items, total, err := h.svc.ListCases(r.Context(), page, perPage)
	if err != nil {
		platform.ServerError(w, r, err)
		return
	}
	platform.JSON(w, http.StatusOK, platform.ListResponse{Items: items, Total: total, Page: page, PerPage: perPage})
}

func (h *Handler) getCase(w http.ResponseWriter, r *http.Request) {
	detail, err := h.svc.GetCase(r.Context(), r.PathValue("caseId"))
	if err != nil {
		respondError(w, r, err)
		return
	}
	platform.JSON(w, http.StatusOK, detail)
}

func (h *Handler) createCase(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := platform.Decode(r, &body); err != nil {
		respondError(w, r, err)
		return
	}
	platform.Trim(&body.Title)
	var v platform.Validator
	v.Required("title", body.Title)
	if err := v.Err(); err != nil {
		respondError(w, r, err)
		return
	}
	c, err := h.svc.CreateCase(r.Context(), body.Title)
	if err != nil {
		respondError(w, r, err)
		return
	}
	platform.JSON(w, http.StatusCreated, c)
}

func (h *Handler) updateCase(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Version *int    `json:"version"`
		Title   *string `json:"title"`
		Status  *string `json:"status"`
	}
	if err := platform.Decode(r, &body); err != nil {
		respondError(w, r, err)
		return
	}
	platform.Trim(body.Title)
	var v platform.Validator
	v.Version(body.Version)
	v.NotBlank("title", body.Title)
	if body.Status != nil {
		v.OneOf("status", *body.Status, validStatuses)
	}
	if err := v.Err(); err != nil {
		respondError(w, r, err)
		return
	}
	c, err := h.svc.UpdateCase(r.Context(), r.PathValue("caseId"), *body.Version, body.Title, body.Status)
	if err != nil {
		respondError(w, r, err)
		return
	}
	platform.JSON(w, http.StatusOK, c)
}

func (h *Handler) deleteCase(w http.ResponseWriter, r *http.Request) {
	version, err := platform.QueryVersion(r)
	if err != nil {
		respondError(w, r, err)
		return
	}
	if err := h.svc.DeleteCase(r.Context(), r.PathValue("caseId"), version); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
