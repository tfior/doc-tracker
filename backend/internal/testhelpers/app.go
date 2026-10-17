// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package testhelpers

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tfior/doc-tracker/internal/app"
	"github.com/tfior/doc-tracker/internal/users"
)

// AppClient sends requests to the fully assembled application, behind the real
// authentication middleware, as a logged-in user.
type AppClient struct {
	UserID  string
	handler http.Handler
	cookies []*http.Cookie
}

// NewAppClient creates a user, logs them in through the login endpoint, and
// returns a client that sends every request with their session cookie.
func NewAppClient(t *testing.T, db *sql.DB) *AppClient {
	t.Helper()
	email := "test-" + randomHex(t) + "@example.com"
	const password = "test-password"
	u, err := users.NewService(users.NewStore(db)).Create(context.Background(), email, "Test", "User", password)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	c := &AppClient{UserID: u.ID, handler: app.NewHandler(db)}
	rec := c.Do("POST", "/api/v1/auth/session", map[string]string{"email": email, "password": password})
	if rec.Code != http.StatusOK {
		t.Fatalf("log in test user: status %d: %s", rec.Code, rec.Body.String())
	}
	c.cookies = rec.Result().Cookies()
	return c
}

// Do sends a request with an optional JSON body and returns the recorded response.
func (c *AppClient) Do(method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, ck := range c.cookies {
		req.AddCookie(ck)
	}
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	return rec
}

// MustCreate sends a POST that must return 201 and returns the new resource's "id".
func (c *AppClient) MustCreate(t *testing.T, path string, body any) string {
	t.Helper()
	rec := c.Do("POST", path, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup: POST %s: status %d: %s", path, rec.Code, rec.Body.String())
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("setup: decode POST %s: %v", path, err)
	}
	return out.ID
}

func randomHex(t *testing.T) string {
	t.Helper()
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random: %v", err)
	}
	return hex.EncodeToString(b)
}

// MustCreateInto sends a POST that must return 201 and decodes the response into out.
func (c *AppClient) MustCreateInto(t *testing.T, path string, body, out any) {
	t.Helper()
	rec := c.Do("POST", path, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup: POST %s: status %d: %s", path, rec.Code, rec.Body.String())
	}
	if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
		t.Fatalf("setup: decode POST %s: %v", path, err)
	}
}

// MustDelete sends a DELETE that must return 204.
func (c *AppClient) MustDelete(t *testing.T, path string) {
	t.Helper()
	if rec := c.Do("DELETE", path, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("setup: DELETE %s: status %d: %s", path, rec.Code, rec.Body.String())
	}
}
