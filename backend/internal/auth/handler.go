package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/tfior/doc-tracker/platform"
)

const cookieName = "session_token"

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/session", h.login)
	mux.HandleFunc("DELETE /api/v1/auth/session", h.logout)
	mux.HandleFunc("GET /api/v1/auth/session", h.getSession)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if body.Email == "" || body.Password == "" {
		platform.Error(w, http.StatusBadRequest, "invalid_input", "email and password are required")
		return
	}

	token, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		platform.Error(w, http.StatusUnauthorized, "unauthorized", "invalid email or password")
		return
	}
	if err != nil {
		platform.Error(w, http.StatusInternalServerError, "internal", "unexpected error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(sessionTTL),
	})
	platform.JSON(w, http.StatusOK, map[string]any{"authenticated": true})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieName)
	if err == nil {
		h.svc.Logout(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	// If we reach here, middleware has already validated the session.
	platform.JSON(w, http.StatusOK, map[string]any{"authenticated": true})
}
