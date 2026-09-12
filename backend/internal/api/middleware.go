package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Platon223/commitin/backend/internal/session"
	"github.com/Platon223/commitin/backend/internal/user"
)

type ctxKey int

const userCtxKey ctxKey = iota

// userFromContext returns the authenticated user attached by requireAuth.
// It panics if called outside a requireAuth-wrapped handler, which is a
// programmer error (a route was registered without the middleware).
func userFromContext(ctx context.Context) *user.User {
	u, ok := ctx.Value(userCtxKey).(*user.User)
	if !ok {
		panic("api: userFromContext called without requireAuth")
	}
	return u
}

// requireAuth validates the Bearer token on the Authorization header, loads
// the corresponding user, and attaches it to the request context. It writes
// a 401 and does not call next if the token is missing, malformed, expired,
// unknown, or its user no longer exists.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing or malformed Authorization header")
			return
		}

		sess, err := s.sessions.Lookup(r.Context(), token)
		if err != nil {
			if errors.Is(err, session.ErrNotFound) {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			writeError(w, http.StatusInternalServerError, "auth check failed")
			return
		}

		u, err := s.users.GetByID(r.Context(), sess.UserID)
		if err != nil {
			if errors.Is(err, user.ErrNotFound) {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			writeError(w, http.StatusInternalServerError, "auth check failed")
			return
		}

		ctx := context.WithValue(r.Context(), userCtxKey, u)
		next(w, r.WithContext(ctx))
	}
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header.
func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}
