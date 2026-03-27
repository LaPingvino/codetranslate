package translator

import "context"

// Translator sends source code to an LLM and gets back translated code.
type Translator interface {
	// Translate converts source code from one language to another.
	Translate(ctx context.Context, req *Request) (*Response, error)

	// Name returns the model/backend name for ledger tracking.
	Name() string
}

type Request struct {
	SourceCode  string
	SourceLang  string
	TargetLang  string
	Context     string // already-translated dependency code
	Conventions string // target language conventions/notes
	LastError   string // compiler error from previous attempt (for retry)
}

type Response struct {
	Code  string
	Model string
}

// ExtractCode pulls code from a response that may contain markdown fences.
func ExtractCode(raw string) string {
	// Look for ```lang ... ``` blocks
	lines := splitLines(raw)
	var code []string
	inBlock := false
	for _, line := range lines {
		if !inBlock && len(line) >= 3 && line[:3] == "```" {
			inBlock = true
			continue
		}
		if inBlock && line == "```" {
			inBlock = false
			continue
		}
		if inBlock {
			code = append(code, line)
		}
	}
	if len(code) > 0 {
		return joinLines(code)
	}
	return raw // no fences found, return as-is
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for i, l := range lines {
		if i > 0 {
			result += "\n"
		}
		result += l
	}
	return result
}
