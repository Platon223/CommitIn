// Package claude judges a commit message against its diff via the Claude
// API, returning a structured verdict.
package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Model is deliberately cheap: this call runs on every commit, billed to the
// user's own key, and judging a short message + diff against a rubric is a
// classification-style task Haiku handles well.
const Model = "claude-haiku-4-5"

const maxTokens = 1024

// PassingScore is the minimum score (out of 10) a commit message needs to
// pass. This is the single source of truth for the good/bad threshold --
// the model only returns a score; Go decides what counts as good, so the
// two can never disagree.
const PassingScore = 6

// Verdict is the judge's structured response.
type Verdict struct {
	Score      int    `json:"score"`
	Roast      string `json:"roast"`
	Suggestion string `json:"suggestion"`
}

// Good reports whether the commit message passed.
func (v Verdict) Good() bool {
	return v.Score >= PassingScore
}

// Client judges commit messages via the Claude API.
type Client struct {
	api anthropic.Client
}

// New builds a Client for apiKey. baseURL overrides the API endpoint for
// tests; pass "" in production to use the real API.
func New(apiKey, baseURL string) *Client {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &Client{api: anthropic.NewClient(opts...)}
}

const toolName = "submit_verdict"

var verdictTool = anthropic.ToolParam{
	Name:        toolName,
	Description: anthropic.String("Submit your judgment of the commit message."),
	Strict:      anthropic.Bool(true),
	InputSchema: anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"score": map[string]any{
				"type":        "integer",
				"minimum":     0,
				"maximum":     10,
				"description": "quality score from 0 (lazy/useless) to 10 (excellent)",
			},
			"roast": map[string]any{
				"type":        "string",
				"description": "a short, honest reaction to the message -- funny and a little savage if it's bad, genuine praise if it's good. Specific to what the diff actually shows, not generic.",
			},
			"suggestion": map[string]any{
				"type":        "string",
				"description": "a better commit message for this diff, imperative mood, usable as-is",
			},
		},
		Required:    []string{"score", "roast", "suggestion"},
		ExtraFields: map[string]any{"additionalProperties": false},
	},
}

const systemPrompt = `You are CommitIn, a tool that judges git commit messages against the actual code change.

Score the given commit message against the given diff, 0-10:
- If the message is lazy, vague, or doesn't reflect what the diff actually does ("fix stuff", "wip", "updates"), score it low (below 6) and roast it: funny, a little savage, but not cruel, and specific to what the diff actually shows.
- If the message clearly and accurately describes the change, score it 6 or above and give genuine, specific praise.
- Always suggest a better commit message for the diff, even when the original already scores well.

Always respond by calling the ` + toolName + ` tool.`

// Judge scores a commit message against its diff. Any request or response
// failure (network, auth, rate limit, malformed response) is returned as an
// error -- it's the caller's decision whether that means failing open or
// closed.
func (c *Client) Judge(ctx context.Context, message, diff string) (*Verdict, error) {
	userContent := fmt.Sprintf("Commit message:\n%s\n\nDiff:\n%s", message, diff)

	resp, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:      Model,
		MaxTokens:  maxTokens,
		System:     []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages:   []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(userContent))},
		Tools:      []anthropic.ToolUnionParam{{OfTool: &verdictTool}},
		ToolChoice: anthropic.ToolChoiceParamOfTool(toolName),
	})
	if err != nil {
		return nil, fmt.Errorf("claude: request failed: %w", err)
	}

	for _, block := range resp.Content {
		tu, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok || tu.Name != toolName {
			continue
		}
		var v Verdict
		if err := json.Unmarshal(tu.Input, &v); err != nil {
			return nil, fmt.Errorf("claude: parse verdict: %w", err)
		}
		if v.Score < 0 || v.Score > 10 {
			return nil, fmt.Errorf("claude: verdict score %d out of range", v.Score)
		}
		if v.Roast == "" || v.Suggestion == "" {
			return nil, errors.New("claude: verdict missing roast or suggestion")
		}
		return &v, nil
	}
	return nil, errors.New("claude: response had no verdict")
}
