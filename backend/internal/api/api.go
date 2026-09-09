// Package api wires the backend HTTP handlers.
package api

import (
	"log"
	"net/http"
	"time"

	"github.com/Platon223/commitin/backend/internal/session"
	"github.com/Platon223/commitin/backend/internal/user"
)

// Server holds the dependencies shared by the HTTP handlers.
type Server struct {
	users    *user.Store
	sessions *session.Store
}

// NewServer builds a Server.
func NewServer(users *user.Store, sessions *session.Store) *Server {
	return &Server{users: users, sessions: sessions}
}

// Routes returns the HTTP handler for the whole API.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /signup", s.handleSignup)
	mux.HandleFunc("POST /login", s.handleLogin)
	return recoverAndLog(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// recoverAndLog logs each request and turns a handler panic into a 500 instead
// of crashing the server.
func recoverAndLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("api: panic on %s %s: %v", r.Method, r.URL.Path, rec)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
