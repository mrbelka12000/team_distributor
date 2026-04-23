package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"github.com/mrbelka12000/team_distributor/internal/models"
)

// ErrNoMembers is returned when the OpenAI model reports that the request
// contained no recognizable players.
var ErrNoMembers = errors.New("openai: no members provided")

const systemPrompt = `You distribute players into balanced teams based on their ratings.

Parse the user's raw message to extract:
- Players with ratings in any format ("Name — 1300", "Name - 1300", "Name 1300", etc.).
- Desired team count if stated (e.g. "2 teams", "3 команды"); default to 2 when not specified.

Rules:
- Each player appears in exactly one team.
- Minimize the absolute difference between team total ratings.
- Keep team sizes as equal as possible (difference of at most 1).
- Give each team a short creative name.
- Preserve player names exactly as written in the input.
- If the input contains no recognizable players with ratings, respond with {"teams":[],"error":"no_members"} and nothing else.

Respond with JSON in this exact shape and nothing else:
{"teams":[{"name":"Name","total":1234,"members":[{"name":"Player","rating":1000}]}]}`

type Client struct {
	c     *openai.Client
	model string
}

func New(apiKey, model string) *Client {
	if model == "" {
		model = openai.GPT4oMini
	}
	slog.Debug("openai client created", "model", model)
	return &Client{
		c:     openai.NewClient(apiKey),
		model: model,
	}
}

func (c *Client) Model() string {
	return c.model
}

// Ping verifies the API key is valid and the configured model is reachable.
func (c *Client) Ping(ctx context.Context) error {
	slog.Debug("openai ping start", "model", c.model)
	start := time.Now()
	if _, err := c.c.GetModel(ctx, c.model); err != nil {
		slog.Error("openai ping failed", "model", c.model, "error", err, "elapsed", time.Since(start))
		return fmt.Errorf("ping model %q: %w", c.model, err)
	}
	slog.Debug("openai ping ok", "model", c.model, "elapsed", time.Since(start))
	return nil
}

type distributeResponse struct {
	Teams []models.Team `json:"teams"`
	Error string        `json:"error"`
}

// Distribute forwards the raw user message plus optional game context to OpenAI
// and parses the JSON team response.
func (c *Client) Distribute(ctx context.Context, rawRequest string, game models.Game) ([]models.Team, error) {
	system := systemPrompt
	if hint := gameHint(game); hint != "" {
		system += "\n\n" + hint
	}

	slog.Info("openai distribute request",
		"model", c.model,
		"game", game.Name,
		"team_size", game.TeamSize,
		"request_chars", len(rawRequest),
	)
	start := time.Now()

	resp, err := c.c.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: system},
			{Role: openai.ChatMessageRoleUser, Content: rawRequest},
		},
	})
	if err != nil {
		slog.Error("openai distribute failed", "error", err, "elapsed", time.Since(start))
		return nil, fmt.Errorf("openai chat completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		slog.Error("openai returned no choices")
		return nil, fmt.Errorf("openai: empty response")
	}

	content := resp.Choices[0].Message.Content
	slog.Debug("openai distribute response",
		"elapsed", time.Since(start),
		"tokens_in", resp.Usage.PromptTokens,
		"tokens_out", resp.Usage.CompletionTokens,
		"response_chars", len(content),
	)

	var out distributeResponse
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		slog.Error("openai json parse failed", "error", err, "body", content)
		return nil, fmt.Errorf("parse openai response: %w", err)
	}
	if out.Error == "no_members" || len(out.Teams) == 0 {
		slog.Info("openai reported no members")
		return nil, ErrNoMembers
	}

	slog.Info("openai distribute ok", "teams", len(out.Teams), "elapsed", time.Since(start))
	return out.Teams, nil
}

func gameHint(game models.Game) string {
	if game.ID == "" {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Game context: %s.", game.Name)
	if game.TeamSize > 0 {
		fmt.Fprintf(&b, " Prefer teams of %d players when the roster allows it.", game.TeamSize)
	}
	return b.String()
}
