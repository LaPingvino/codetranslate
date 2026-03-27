package compiler

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Compiler checks whether the target code compiles.
type Compiler interface {
	Compile(ctx context.Context, dir string) error
}

// Linter runs lightweight syntax/lint checks without full compilation.
type Linter struct {
	Lang string
}

func (l *Linter) Compile(ctx context.Context, dir string) error {
	switch {
	case strings.Contains(strings.ToLower(l.Lang), "go"):
		cmd := exec.CommandContext(ctx, "go", "vet", "./...")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("go vet failed:\n%s", strings.TrimSpace(string(out)))
		}
	case strings.Contains(strings.ToLower(l.Lang), "c"):
		// Use gcc -fsyntax-only for quick syntax check
		cmd := exec.CommandContext(ctx, "sh", "-c",
			fmt.Sprintf("find %s -name '*.c' -exec gcc -fsyntax-only -Wall {} +", dir))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("lint failed:\n%s", strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// ForLint returns a lightweight linter for the given language.
func ForLint(lang string) Compiler {
	return &Linter{Lang: lang}
}

// GoCompiler compiles Go code.
type GoCompiler struct{}

func (g *GoCompiler) Compile(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed:\n%s", strings.TrimSpace(string(out)))
	}
	return nil
}

// CCompiler compiles C/C++ code using make or gcc.
type CCompiler struct {
	UseMake bool
	CC      string // e.g. "gcc", "g++", "cc"
}

func (c *CCompiler) Compile(ctx context.Context, dir string) error {
	if c.UseMake {
		cmd := exec.CommandContext(ctx, "make", "-C", dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("make failed:\n%s", strings.TrimSpace(string(out)))
		}
		return nil
	}
	cc := c.CC
	if cc == "" {
		cc = "gcc"
	}
	cmd := exec.CommandContext(ctx, cc, "-c", "-fsyntax-only", "*.c")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed:\n%s", cc, strings.TrimSpace(string(out)))
	}
	return nil
}

// ForLanguage returns a compiler for the given target language.
// ForLanguage returns a compiler for the given target language.
// Matches flexibly: "c", "modern C (C11)", "C++", etc.
func ForLanguage(lang string) Compiler {
	lower := strings.ToLower(lang)
	switch {
	case lower == "go" || lower == "golang":
		return &GoCompiler{}
	case strings.Contains(lower, "c++") || strings.Contains(lower, "cpp"):
		return &CCompiler{CC: "g++", UseMake: true}
	case strings.Contains(lower, "c"):
		return &CCompiler{CC: "gcc", UseMake: true}
	default:
		return &CCompiler{CC: "gcc", UseMake: true}
	}
}
