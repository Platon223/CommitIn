package api

import (
	"errors"
	"net/http"

	"github.com/Platon223/commitin/backend/internal/score"
)

type submitScoreRequest struct {
	Score     int    `json:"score"`
	AttemptID string `json:"attempt_id"`
	RepoName  string `json:"repo_name"`
}

func (s *Server) handleSubmitScore(w http.ResponseWriter, r *http.Request) {
	var req submitScoreRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Score < score.MinScore || req.Score > score.MaxScore {
		writeError(w, http.StatusBadRequest, "score must be between 0 and 10")
		return
	}
	if req.AttemptID == "" {
		writeError(w, http.StatusBadRequest, "attempt_id is required")
		return
	}
	if req.RepoName == "" {
		writeError(w, http.StatusBadRequest, "repo_name is required")
		return
	}

	u := userFromContext(r.Context())
	doc, err := s.scores.Create(r.Context(), u.ID, req.Score, req.AttemptID, req.RepoName)
	if err != nil {
		switch {
		case errors.Is(err, score.ErrDuplicate):
			writeError(w, http.StatusConflict, "score already submitted for this attempt")
		case errors.Is(err, score.ErrRateLimited):
			writeError(w, http.StatusTooManyRequests, "too many score submissions, try again later")
		default:
			writeError(w, http.StatusInternalServerError, "could not save score")
		}
		return
	}

	writeJSON(w, http.StatusCreated, doc)
}

// handleLeaderboard is public -- no auth -- per the product's "public
// leaderboard" concept.
func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := s.scores.Leaderboard(r.Context(), score.LeaderboardWindow, score.LeaderboardMinCommits, score.LeaderboardLimit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load leaderboard")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"leaderboard": entries,
		"window_days": int(score.LeaderboardWindow.Hours() / 24),
		"min_commits": score.LeaderboardMinCommits,
	})
}

// handleStats returns the caller's own history -- never anyone else's.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r.Context())
	st, err := s.scores.Stats(r.Context(), u.ID, score.StatsWindow)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load stats")
		return
	}
	writeJSON(w, http.StatusOK, st)
}
