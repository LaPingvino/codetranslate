package translator

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Command implements Translator by shelling out to any CLI command.
// The command receives the prompt on stdin and should output translated code on stdout.
// Example usage: --model "command:claude -p" or --model "command:ollama run codellama"
type Command struct {
	Cmd  string   // the command to run
	Args []string // arguments
}

func NewCommand(cmdLine string) *Command {
	parts := strings.Fields(cmdLine)
	if len(parts) == 0 {
		return &Command{Cmd: "cat"} // fallback: identity
	}
	return &Command{Cmd: parts[0], Args: parts[1:]}
}

func (c *Command) Name() string {
	return "command:" + c.Cmd
}

func (c *Command) Translate(ctx context.Context, req *Request) (*Response, error) {
	prompt := buildPrompt(req)

	cmd := exec.CommandContext(ctx, c.Cmd, c.Args...)
	cmd.Stdin = strings.NewReader(prompt)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s failed: %s\nstderr: %s", c.Cmd, err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("%s failed: %w", c.Cmd, err)
	}

	code := ExtractCode(strings.TrimSpace(string(out)))
	return &Response{Code: code, Model: c.Name()}, nil
}
