package translator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Manual implements Translator by writing prompts to files for human translation.
type Manual struct {
	OutputDir string
}

func NewManual(outputDir string) *Manual {
	return &Manual{OutputDir: outputDir}
}

func (m *Manual) Name() string {
	return "manual"
}

func (m *Manual) Translate(_ context.Context, req *Request) (*Response, error) {
	if err := os.MkdirAll(m.OutputDir, 0755); err != nil {
		return nil, err
	}

	prompt := buildPrompt(req)

	// Write the prompt to a file for the human
	promptFile := filepath.Join(m.OutputDir, "prompt.txt")
	if err := os.WriteFile(promptFile, []byte(prompt), 0644); err != nil {
		return nil, err
	}

	// Check if a response file exists
	responseFile := filepath.Join(m.OutputDir, "response.txt")
	if data, err := os.ReadFile(responseFile); err == nil && len(data) > 0 {
		// Clean up response file after reading
		os.Remove(responseFile)
		return &Response{Code: string(data), Model: "manual"}, nil
	}

	return nil, fmt.Errorf("manual translation needed: write your translation to %s", responseFile)
}
