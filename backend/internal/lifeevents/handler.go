package lifeevents

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tfior/doc-tracker/internal/activitylog"
	"github.com/tfior/doc-tracker/internal/auth"
	"github.com/tfior/doc-tracker/platform"
)

var validEventTypes = map[string]bool{
	"birth": true, "marriage": true, "death": true,
	"naturalization": true, "immigration": true, "other": true,
}

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
	mux.HandleFunc("GET /api/v1/cases/{caseId}/life-events", h.listLifeEvents)
	mux.HandleFunc("POST /api/v1/cases/{caseId}/life-events", h.createLifeEvent)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}/life-events/{eventId}", h.updateLifeEvent)
	mux.HandleFunc("DELETE /api/v1/cases/{caseId}/life-events/{eventId}", h.deleteLifeEvent)
	mux.HandleFunc("PATCH /api/v1/cases/{caseId}/life-events/{eventId}/person", h.reassignLifeEvent)
}

func (h *Handler) listLifeEvents(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")

	page, perPage, ok := platform.ParsePagination(w, r)
	if !ok {
		return
	}

	items, total, err := h.svc.ListLifeEvents(r.Context(), caseID, page, perPage)
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

func (h *Handler) createLifeEvent(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")

	var body struct {
		PersonID         string              `json:"person_id"`
		EventType        string              `json:"event_type"`
		EventDate        platform.NullString `json:"event_date"`
		EventPlace       platform.NullString `json:"event_place"`
		SpouseName       platform.NullString `json:"spouse_name"`
		SpouseBirthDate  platform.NullString `json:"spouse_birth_date"`
		SpouseBirthPlace platform.NullString `json:"spouse_birth_place"`
		Notes            platform.NullString `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.PersonID == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "person_id is required")
		return
	}
	if body.EventType == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "event_type is required")
		return
	}
	if !validEventTypes[body.EventType] {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid event_type")
		return
	}

	input := UpdateLifeEventInput{
		EventDate:        fromPlatformNull(body.EventDate),
		EventPlace:       fromPlatformNull(body.EventPlace),
		SpouseName:       fromPlatformNull(body.SpouseName),
		SpouseBirthDate:  fromPlatformNull(body.SpouseBirthDate),
		SpouseBirthPlace: fromPlatformNull(body.SpouseBirthPlace),
		Notes:            fromPlatformNull(body.Notes),
	}

	le, err := h.svc.CreateLifeEvent(r.Context(), caseID, body.PersonID, body.EventType, input)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Person not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	h.log(r, activitylog.InsertParams{
		CaseID: caseID, Action: "created", EntityType: "life_event",
		EntityID: le.ID, EntityName: le.EventType + " event",
	})
	platform.JSON(w, http.StatusCreated, le)
}

func (h *Handler) updateLifeEvent(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	eventID := r.PathValue("eventId")

	var body struct {
		EventType        *string             `json:"event_type"`
		EventDate        platform.NullString `json:"event_date"`
		EventPlace       platform.NullString `json:"event_place"`
		SpouseName       platform.NullString `json:"spouse_name"`
		SpouseBirthDate  platform.NullString `json:"spouse_birth_date"`
		SpouseBirthPlace platform.NullString `json:"spouse_birth_place"`
		Notes            platform.NullString `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.EventType != nil && !validEventTypes[*body.EventType] {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid event_type")
		return
	}

	input := UpdateLifeEventInput{
		EventType:        body.EventType,
		EventDate:        fromPlatformNull(body.EventDate),
		EventPlace:       fromPlatformNull(body.EventPlace),
		SpouseName:       fromPlatformNull(body.SpouseName),
		SpouseBirthDate:  fromPlatformNull(body.SpouseBirthDate),
		SpouseBirthPlace: fromPlatformNull(body.SpouseBirthPlace),
		Notes:            fromPlatformNull(body.Notes),
	}

	le, err := h.svc.UpdateLifeEvent(r.Context(), caseID, eventID, input)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Life event not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	var changes []activitylog.FieldChange
	if body.EventType != nil {
		changes = append(changes, activitylog.FieldChange{Field: "event_type", To: *body.EventType})
	}
	for _, f := range []struct {
		ns    platform.NullString
		field string
	}{
		{body.EventDate, "event_date"}, {body.EventPlace, "event_place"},
		{body.SpouseName, "spouse_name"}, {body.SpouseBirthDate, "spouse_birth_date"},
		{body.SpouseBirthPlace, "spouse_birth_place"}, {body.Notes, "notes"},
	} {
		if f.ns.Set {
			var val interface{}
			if f.ns.Valid {
				val = f.ns.Value
			}
			changes = append(changes, activitylog.FieldChange{Field: f.field, To: val})
		}
	}
	h.log(r, activitylog.InsertParams{
		CaseID: caseID, Action: "updated", EntityType: "life_event",
		EntityID: le.ID, EntityName: le.EventType + " event", Changes: changes,
	})
	platform.JSON(w, http.StatusOK, le)
}

func (h *Handler) deleteLifeEvent(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	eventID := r.PathValue("eventId")

	err := h.svc.DeleteLifeEvent(r.Context(), caseID, eventID)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Life event not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	h.log(r, activitylog.InsertParams{
		CaseID: caseID, Action: "deleted", EntityType: "life_event",
		EntityID: eventID, EntityName: eventID,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reassignLifeEvent(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("caseId")
	eventID := r.PathValue("eventId")

	var body struct {
		PersonID string `json:"person_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.PersonID == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "person_id is required")
		return
	}

	le, err := h.svc.ReassignLifeEvent(r.Context(), caseID, eventID, body.PersonID)
	if errors.Is(err, ErrNotFound) {
		platform.Error(w, http.StatusNotFound, "not_found", "Life event or person not found")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	h.log(r, activitylog.InsertParams{
		CaseID: caseID, Action: "updated", EntityType: "life_event",
		EntityID: le.ID, EntityName: le.EventType + " event",
		Changes: []activitylog.FieldChange{{Field: "person_id", To: body.PersonID}},
	})
	platform.JSON(w, http.StatusOK, le)
}

func fromPlatformNull(n platform.NullString) NullableField {
	return NullableField{Set: n.Set, Valid: n.Valid, Value: n.Value}
}
