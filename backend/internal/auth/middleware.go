package auth

import (
	"context"
	"net/http"

	"github.com/tfior/doc-tracker/platform"
)

type contextKey string

const contextKeyUserID contextKey = "user_id"

// Middleware returns an HTTP middleware that requires a valid session cookie on
// all routes except POST /api/v1/auth/session (login) and GET /health.
// On success it injects the authenticated user's ID into the request context.
func Middleware(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || (r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/session") {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(cookieName)
			if err != nil {
				platform.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}

			sess, ok := svc.GetSession(cookie.Value)
			if !ok {
				platform.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID, sess.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the authenticated user's ID from the request context.
// Returns an empty string and false if not present.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(contextKeyUserID).(string)
	return id, ok
}
