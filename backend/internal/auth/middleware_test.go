package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tfior/doc-tracker/internal/auth"
	"github.com/tfior/doc-tracker/internal/users"
)

// newMiddlewareEnv wires up a middleware with a pre-seeded session token.
// The users service is not exercised by the middleware, so a nil DB is safe here.
func newMiddlewareEnv(t *testing.T) (handler http.Handler, validToken string) {
	t.Helper()

	store := auth.NewSessionStore()
	validToken = "test-valid-token"
	store.Create(validToken, "test-user-id", time.Hour)

	svc := auth.NewService(store, users.NewService(users.NewStore(nil)))

	sentinel := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return auth.Middleware(svc)(sentinel), validToken
}

func TestMiddleware_AuthenticatedRequestPassesThrough(t *testing.T) {
	handler, token := newMiddlewareEnv(t)

	req := httptest.NewRequest("GET", "/api/v1/cases", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMiddleware_UnauthenticatedRequestReturns401(t *testing.T) {
	handler, _ := newMiddlewareEnv(t)

	req := httptest.NewRequest("GET", "/api/v1/cases", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_LoginEndpointBypassesAuth(t *testing.T) {
	handler, _ := newMiddlewareEnv(t)

	// No cookie — the login endpoint must not be blocked.
	req := httptest.NewRequest("POST", "/api/v1/auth/session", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddleware_HealthEndpointBypassesAuth(t *testing.T) {
	handler, _ := newMiddlewareEnv(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
}
