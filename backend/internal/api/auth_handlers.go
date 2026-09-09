package api

import (
	"errors"
	"net/http"

	"github.com/Platon223/commitin/backend/internal/auth"
	"github.com/Platon223/commitin/backend/internal/user"
)

// authResponse is returned by both /signup and /login.
type authResponse struct {
	Token string     `json:"token"`
	User  *user.User `json:"user"`
}

type signupRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := user.ValidateEmail(req.Email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := user.ValidateUsername(req.Username); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := user.ValidatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not process password")
		return
	}

	u := &user.User{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hash,
	}
	if err := s.users.Create(r.Context(), u); err != nil {
		if errors.Is(err, user.ErrDuplicate) {
			writeError(w, http.StatusConflict, "email or username already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create account")
		return
	}

	token, _, err := s.sessions.Create(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: u})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	u, err := s.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			auth.CheckDummy(req.Password) // equalize timing with the found path
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, _, err := s.sessions.Create(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, User: u})
}
