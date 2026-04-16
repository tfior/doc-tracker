package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tfior/doc-tracker/internal/auth"
	"github.com/tfior/doc-tracker/internal/testhelpers"
	"github.com/tfior/doc-tracker/internal/users"
)

// testEnv holds the wired-up services and a ready-to-use mux for one test run.
type testEnv struct {
	authSvc *auth.Service
	userSvc *users.Service
	mux     *http.ServeMux
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	testhelpers.TruncateUsers(t, db)

	userSvc := users.NewService(users.NewStore(db))
	sessionStore := auth.NewSessionStore()
	authSvc := auth.NewService(sessionStore, userSvc)

	mux := http.NewServeMux()
	auth.NewHandler(authSvc).RegisterRoutes(mux)

	return &testEnv{authSvc: authSvc, userSvc: userSvc, mux: mux}
}

func (e *testEnv) createUser(t *testing.T, email, password string) *users.User {
	t.Helper()
	u, err := e.userSvc.Create(context.Background(), email, "Test", "User", password)
	require.NoError(t, err)
	return u
}

func (e *testEnv) do(method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) doWithCookie(method, path, cookieValue string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: cookieValue})
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_token" {
			return c
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// POST /api/v1/auth/session — login
// ---------------------------------------------------------------------------

func TestLogin_ValidCredentials(t *testing.T) {
	env := newTestEnv(t)
	env.createUser(t, "user@example.com", "secret123")

	rec := env.do("POST", "/api/v1/auth/session", map[string]string{
		"email": "user@example.com", "password": "secret123",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	cookie := sessionCookie(rec)
	require.NotNil(t, cookie, "session_token cookie should be set")
	assert.NotEmpty(t, cookie.Value)
	assert.True(t, cookie.HttpOnly)
}

func TestLogin_WrongPassword(t *testing.T) {
	env := newTestEnv(t)
	env.createUser(t, "user@example.com", "secret123")

	rec := env.do("POST", "/api/v1/auth/session", map[string]string{
		"email": "user@example.com", "password": "wrongpassword",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Nil(t, sessionCookie(rec))
}

func TestLogin_UnknownEmail(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("POST", "/api/v1/auth/session", map[string]string{
		"email": "nobody@example.com", "password": "secret123",
	})

	// Same 401 as wrong password — no user enumeration
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Nil(t, sessionCookie(rec))
}

func TestLogin_MissingEmail(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("POST", "/api/v1/auth/session", map[string]string{
		"password": "secret123",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_MissingPassword(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("POST", "/api/v1/auth/session", map[string]string{
		"email": "user@example.com",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_EmptyBody(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("POST", "/api/v1/auth/session", nil)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/auth/session — logout
// ---------------------------------------------------------------------------

func TestLogout_ValidSession(t *testing.T) {
	env := newTestEnv(t)
	env.createUser(t, "user@example.com", "secret123")

	loginRec := env.do("POST", "/api/v1/auth/session", map[string]string{
		"email": "user@example.com", "password": "secret123",
	})
	require.Equal(t, http.StatusOK, loginRec.Code)
	token := sessionCookie(loginRec).Value

	rec := env.doWithCookie("DELETE", "/api/v1/auth/session", token)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	cleared := sessionCookie(rec)
	require.NotNil(t, cleared, "session_token cookie should be present to clear it")
	assert.Empty(t, cleared.Value)
	assert.True(t, cleared.MaxAge < 0, "MaxAge should be negative to delete the cookie")
}

func TestLogout_NoSession(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do("DELETE", "/api/v1/auth/session", nil)

	// Logout is idempotent — no session cookie is not an error
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// ---------------------------------------------------------------------------
// GET /api/v1/auth/session — session check
// ---------------------------------------------------------------------------

func TestGetSession_ValidSession(t *testing.T) {
	env := newTestEnv(t)
	env.createUser(t, "user@example.com", "secret123")

	loginRec := env.do("POST", "/api/v1/auth/session", map[string]string{
		"email": "user@example.com", "password": "secret123",
	})
	require.Equal(t, http.StatusOK, loginRec.Code)
	token := sessionCookie(loginRec).Value

	// Wrap with middleware so GET /api/v1/auth/session is protected
	protected := http.NewServeMux()
	auth.NewHandler(env.authSvc).RegisterRoutes(protected)
	handler := auth.Middleware(env.authSvc)(protected)

	req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetSession_NoCookie(t *testing.T) {
	env := newTestEnv(t)

	protected := auth.Middleware(env.authSvc)(env.mux)
	req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetSession_InvalidToken(t *testing.T) {
	env := newTestEnv(t)

	protected := auth.Middleware(env.authSvc)(env.mux)
	req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "not-a-real-token"})
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
