package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "translate",
	Short: "Systematic codebase translation between programming languages",
	Long: `codetranslate - LLM-agnostic framework for systematic codebase translation.

The framework manages inventory, scheduling, and verification.
The LLM does the translation. The compiler is the judge.`,
}

func Execute() error {
	return rootCmd.Execute()
}
