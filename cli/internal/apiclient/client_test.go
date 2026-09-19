package apiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(srv.URL)
}

func TestAPIErrorKeepsBackendMessageAndStatus(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid email or password"}`))
	})

	_, err := c.Login(context.Background(), "a@b.c", "nope")
	if err == nil {
		t.Fatal("expected an error")
	}
	// Existing callers just print err.Error(); that text must not change.
	if err.Error() != "invalid email or password" {
		t.Fatalf("Error() = %q, want the backend's message verbatim", err.Error())
	}
	if !IsUnauthorized(err) {
		t.Fatal("a 401 should satisfy IsUnauthorized")
	}
}

func TestAPIErrorWithoutBodyFallsBackToStatus(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.Leaderboard(context.Background())
	if err == nil || IsUnauthorized(err) {
		t.Fatalf("want a non-401 error, got %v", err)
	}
	if got := err.Error(); got != "request failed: 500 Internal Server Error" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestIsUnauthorizedIgnoresOtherErrors(t *testing.T) {
	if IsUnauthorized(nil) {
		t.Fatal("nil is not unauthorized")
	}
	if IsUnauthorized(&APIError{StatusCode: http.StatusForbidden, Message: "x"}) {
		t.Fatal("403 is not 401")
	}
}

func TestStatsSendsBearerTokenAndDecodes(t *testing.T) {
	var gotAuth, gotPath, gotMethod string
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath, gotMethod = r.Header.Get("Authorization"), r.URL.Path, r.Method
		w.Write([]byte(`{"window_days":30,"passing_score":6,
			"current":{"attempts":8,"average_score":7,"rejected":2},
			"previous":{"attempts":3,"average_score":5.33,"rejected":2},
			"daily":[{"date":"2026-09-19","attempts":2,"average_score":6.5}]}`))
	})

	st, err := c.Stats(context.Background(), "tok123")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || gotPath != "/stats" || gotAuth != "Bearer tok123" {
		t.Fatalf("request was %s %s with Authorization %q", gotMethod, gotPath, gotAuth)
	}
	if st.WindowDays != 30 || st.PassingScore != 6 ||
		st.Current.Attempts != 8 || st.Current.AverageScore != 7 || st.Current.Rejected != 2 ||
		st.Previous.Attempts != 3 || st.Previous.AverageScore != 5.33 ||
		len(st.Daily) != 1 || st.Daily[0].Date != "2026-09-19" {
		t.Fatalf("decoded wrongly: %+v", st)
	}
}

func TestLeaderboardStaysUnauthenticated(t *testing.T) {
	var gotAuth string
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"leaderboard":[],"window_days":30,"min_commits":10}`))
	})
	if _, err := c.Leaderboard(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "" {
		t.Fatalf("the public leaderboard must not send credentials, sent %q", gotAuth)
	}
}
