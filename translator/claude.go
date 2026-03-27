package translator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Claude implements Translator using the Anthropic API.
type Claude struct {
	APIKey string
	Model  string // e.g. "claude-haiku-4-5-20251001", "claude-sonnet-4-6-20250514"
}

func NewClaude(model string) *Claude {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	m := resolveClaudeModel(model)
	return &Claude{APIKey: apiKey, Model: m}
}

func (c *Claude) Name() string {
	return c.Model
}

func (c *Claude) Translate(ctx context.Context, req *Request) (*Response, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	prompt := buildPrompt(req)

	body := map[string]interface{}{
		"model":      c.Model,
		"max_tokens": 4096,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	code := ExtractCode(result.Content[0].Text)
	return &Response{Code: code, Model: c.Model}, nil
}

func resolveClaudeModel(shortName string) string {
	switch shortName {
	case "haiku":
		return "claude-haiku-4-5-20251001"
	case "sonnet":
		return "claude-sonnet-4-6-20250514"
	case "opus":
		return "claude-opus-4-6-20250610"
	default:
		return shortName // assume it's a full model ID
	}
}

func buildPrompt(req *Request) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Translate the following %s code to %s.\n\n", req.SourceLang, req.TargetLang))

	if req.Context != "" {
		b.WriteString("Here are already-translated dependencies to reference:\n```")
		b.WriteString(req.TargetLang)
		b.WriteString("\n")
		b.WriteString(req.Context)
		b.WriteString("\n```\n\n")
	}

	if req.Conventions != "" {
		b.WriteString("Target language conventions:\n")
		b.WriteString(req.Conventions)
		b.WriteString("\n\n")
	}

	if req.LastError != "" {
		b.WriteString("A previous translation attempt failed to compile with this error:\n```\n")
		b.WriteString(req.LastError)
		b.WriteString("\n```\nPlease fix the translation.\n\n")
	}

	b.WriteString("Source code:\n```")
	b.WriteString(req.SourceLang)
	b.WriteString("\n")
	b.WriteString(req.SourceCode)
	b.WriteString("\n```\n\n")
	b.WriteString("Output ONLY the translated code. No explanations, no markdown fences.")

	return b.String()
}
