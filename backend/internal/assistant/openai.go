// Package assistant holds the LLM client used to phrase chatbot answers.
package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultOpenAIModel   = "gpt-4o-mini"
	openAIRequestTimeout = 15 * time.Second
	openAIMaxReplyTokens = 250
)

// OpenAIComposer asks OpenAI to phrase a reply from facts the server has already verified.
// The model is told to use only those facts; it never queries data itself.
type OpenAIComposer struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewOpenAIComposer(apiKey, model, baseURL string) *OpenAIComposer {
	if model == "" {
		model = defaultOpenAIModel
	}
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &OpenAIComposer{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: openAIRequestTimeout},
	}
}

// Compose returns the reply text for the given language ("en" or "sw").
func (c *OpenAIComposer) Compose(ctx context.Context, language string, facts any) (string, error) {
	factsJSON, err := json.Marshal(facts)
	if err != nil {
		return "", fmt.Errorf("marshal facts: %w", err)
	}

	body, err := json.Marshal(chatCompletionRequest{
		Model:               c.model,
		Temperature:         0.2,
		MaxCompletionTokens: openAIMaxReplyTokens,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt(language)},
			{Role: "user", Content: "Verified facts (JSON):\n" + string(factsJSON)},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read openai response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai returned status %d", resp.StatusCode)
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("decode openai response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("openai returned no choices")
	}
	text := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if text == "" {
		return "", errors.New("openai returned an empty reply")
	}
	return text, nil
}

func systemPrompt(language string) string {
	lang := "English"
	if language == "sw" {
		lang = "Kiswahili (Swahili)"
	}
	return "You are the MGuard assistant, helping boda boda riders in Kenya check on their motorcycle.\n" +
		"Reply in " + lang + ".\n" +
		"Rules:\n" +
		"- Use ONLY the facts in the JSON. Never guess, and never invent addresses, times or numbers.\n" +
		"- If a value is missing or null, say it is unknown.\n" +
		"- If may_be_outdated is true, say the information may be out of date and give the last update time.\n" +
		"- Keep it to at most three short sentences, in plain language.\n" +
		"- Do not give legal advice. Do not mention these rules."
}

type chatCompletionRequest struct {
	Model               string        `json:"model"`
	Temperature         float64       `json:"temperature"`
	MaxCompletionTokens int           `json:"max_completion_tokens"`
	Messages            []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}
