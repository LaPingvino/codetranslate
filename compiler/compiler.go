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
func ForLanguage(lang string) Compiler {
	switch strings.ToLower(lang) {
	case "go", "golang":
		return &GoCompiler{}
	case "c":
		return &CCompiler{CC: "gcc"}
	case "c++", "cpp":
		return &CCompiler{CC: "g++"}
	default:
		return &GoCompiler{} // fallback
	}
}
