package scanner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/LaPingvino/codetranslate/ledger"
)

// CtagsScanner uses universal-ctags to discover translatable units.
type CtagsScanner struct{}

type ctagsEntry struct {
	Type     string `json:"_type"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Line     int    `json:"line"`
	End      int    `json:"end"`
	Language string `json:"language"`
	Scope    string `json:"scope"`
	ScopeKind string `json:"scopeKind"`
}

func (s *CtagsScanner) Scan(sourceDir, sourceLang, targetLang string) ([]*ledger.Unit, error) {
	// Run ctags with JSON output
	cmd := exec.Command("ctags",
		"--output-format=json",
		"--fields=+neK",
		"--kinds-all=*",
		"-R",
		sourceDir,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ctags failed: %w (is universal-ctags installed?)", err)
	}

	var units []*ledger.Unit
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry ctagsEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip unparseable lines
		}
		if entry.Type != "tag" {
			continue
		}

		kind := normalizeKind(entry.Kind)
		if kind == "" {
			continue // skip unknown kinds
		}

		relPath, _ := filepath.Rel(sourceDir, entry.Path)
		if relPath == "" {
			relPath = entry.Path
		}

		// Read source code for this entry
		sourceCode := extractSource(entry.Path, entry.Line, entry.End)

		u := &ledger.Unit{
			SourceFile: relPath,
			SourceName: entry.Name,
			SourceLang: sourceLang,
			TargetLang: targetLang,
			Kind:       kind,
			Status:     ledger.StatusTodo,
			Tier:       tierForKind(kind),
			SourceCode: sourceCode,
		}

		// Generate target file path (simple mapping)
		u.TargetFile = mapTargetFile(relPath, sourceLang, targetLang)
		u.TargetName = entry.Name // keep same name initially

		units = append(units, u)
	}

	return units, nil
}

func normalizeKind(k string) string {
	switch strings.ToLower(k) {
	case "function", "func", "subroutine":
		return "function"
	case "method", "member":
		return "method"
	case "class", "struct", "structure", "type", "typedef", "union", "enum":
		return "type"
	case "constant", "const", "define", "macro", "enumerator":
		return "const"
	case "variable", "var", "externvar":
		return "var"
	case "interface":
		return "interface"
	case "namespace", "module", "package":
		return "" // skip these
	default:
		return ""
	}
}

func tierForKind(kind string) int {
	switch kind {
	case "const":
		return 0
	case "type", "interface":
		return 1
	case "var":
		return 2
	case "function":
		return 3
	case "method":
		return 4
	default:
		return 5
	}
}

func extractSource(file string, startLine, endLine int) string {
	if startLine <= 0 {
		return ""
	}
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()

	if endLine <= 0 {
		// ctags didn't give us an end line; grab a reasonable chunk
		endLine = startLine + 50
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum >= startLine && lineNum <= endLine {
			lines = append(lines, scanner.Text())
		}
		if lineNum > endLine {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func mapTargetFile(relPath, fromLang, toLang string) string {
	ext := targetExtension(toLang)
	oldExt := filepath.Ext(relPath)
	if oldExt != "" {
		return strings.TrimSuffix(relPath, oldExt) + ext
	}
	return relPath + ext
}

func targetExtension(lang string) string {
	switch strings.ToLower(lang) {
	case "go", "golang":
		return ".go"
	case "c":
		return ".c"
	case "c++", "cpp":
		return ".cpp"
	case "python", "py":
		return ".py"
	case "rust", "rs":
		return ".rs"
	case "java":
		return ".java"
	case "typescript", "ts":
		return ".ts"
	case "javascript", "js":
		return ".js"
	default:
		return ".txt"
	}
}
