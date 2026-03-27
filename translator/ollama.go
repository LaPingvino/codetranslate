package translator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Ollama implements Translator using a local Ollama instance.
type Ollama struct {
	Model   string
	BaseURL string
}

func NewOllama(model string) *Ollama {
	if model == "" {
		model = "llama3"
	}
	return &Ollama{Model: model, BaseURL: "http://localhost:11434"}
}

func (o *Ollama) Name() string {
	return "ollama:" + o.Model
}

func (o *Ollama) Translate(ctx context.Context, req *Request) (*Response, error) {
	prompt := buildPrompt(req)

	body := map[string]interface{}{
		"model":  o.Model,
		"prompt": prompt,
		"stream": false,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.BaseURL+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Ollama error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	code := ExtractCode(result.Response)
	return &Response{Code: code, Model: o.Name()}, nil
}
