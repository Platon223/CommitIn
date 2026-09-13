package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockServer replies to every request with the given status and body.
func mockServer(t *testing.T, status int, body any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Fatal(err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func toolUseResponse(input map[string]any) map[string]any {
	return map[string]any{
		"id":    "msg_test",
		"type":  "message",
		"role":  "assistant",
		"model": Model,
		"content": []map[string]any{
			{"type": "tool_use", "id": "toolu_1", "name": toolName, "input": input},
		},
		"stop_reason":   "tool_use",
		"stop_sequence": nil,
		"usage":         map[string]any{"input_tokens": 10, "output_tokens": 5},
	}
}

func TestJudgeParsesBadVerdict(t *testing.T) {
	srv := mockServer(t, http.StatusOK, toolUseResponse(map[string]any{
		"score":      2,
		"roast":      "groundbreaking work, truly",
		"suggestion": "fix: correct off-by-one error in pagination",
	}))

	c := New("test-key", srv.URL)
	v, err := c.Judge(context.Background(), "fix stuff", "diff --git a/x b/x\n+x\n")
	if err != nil {
		t.Fatalf("Judge: %v", err)
	}
	if v.Good() || v.Score != 2 || v.Roast == "" || v.Suggestion == "" {
		t.Fatalf("unexpected verdict: %+v", v)
	}
}

func TestJudgeParsesGoodVerdict(t *testing.T) {
	srv := mockServer(t, http.StatusOK, toolUseResponse(map[string]any{
		"score":      9,
		"roast":      "clear and specific, nice work",
		"suggestion": "fix: correct off-by-one error in pagination",
	}))

	c := New("test-key", srv.URL)
	v, err := c.Judge(context.Background(), "fix: correct off-by-one error in pagination", "diff --git a/x b/x\n+x\n")
	if err != nil {
		t.Fatalf("Judge: %v", err)
	}
	if !v.Good() || v.Score != 9 {
		t.Fatalf("unexpected verdict: %+v", v)
	}
}

func TestVerdictGoodThreshold(t *testing.T) {
	if (Verdict{Score: PassingScore - 1}).Good() {
		t.Fatalf("score %d should not be good", PassingScore-1)
	}
	if !(Verdict{Score: PassingScore}).Good() {
		t.Fatalf("score %d should be good", PassingScore)
	}
}

func TestJudgeNoToolUseBlock(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]any{
		"id": "msg_test", "type": "message", "role": "assistant", "model": Model,
		"content":       []map[string]any{{"type": "text", "text": "I refuse to use the tool."}},
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage":         map[string]any{"input_tokens": 10, "output_tokens": 5},
	})

	c := New("test-key", srv.URL)
	if _, err := c.Judge(context.Background(), "msg", "diff"); err == nil {
		t.Fatal("expected an error when the response has no tool_use block")
	}
}

func TestJudgeScoreOutOfRange(t *testing.T) {
	srv := mockServer(t, http.StatusOK, toolUseResponse(map[string]any{
		"score": 42, "roast": "x", "suggestion": "y",
	}))

	c := New("test-key", srv.URL)
	if _, err := c.Judge(context.Background(), "msg", "diff"); err == nil {
		t.Fatal("expected an error for an out-of-range score")
	}
}

func TestJudgeMissingRoastOrSuggestion(t *testing.T) {
	srv := mockServer(t, http.StatusOK, toolUseResponse(map[string]any{
		"score": 3, "roast": "", "suggestion": "",
	}))

	c := New("test-key", srv.URL)
	if _, err := c.Judge(context.Background(), "msg", "diff"); err == nil {
		t.Fatal("expected an error for a verdict missing roast/suggestion")
	}
}

func TestJudgeAPIError(t *testing.T) {
	srv := mockServer(t, http.StatusTooManyRequests, map[string]any{
		"type":  "error",
		"error": map[string]any{"type": "rate_limit_error", "message": "rate limited"},
	})

	c := New("test-key", srv.URL)
	if _, err := c.Judge(context.Background(), "msg", "diff"); err == nil {
		t.Fatal("expected an error on a 429 response")
	}
}
