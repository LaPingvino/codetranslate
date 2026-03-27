package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/LaPingvino/codetranslate/config"
	"github.com/LaPingvino/codetranslate/ledger"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show Dolt diff of the translation ledger",
	RunE:  runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	root, err := config.ProjectRoot()
	if err != nil {
		return err
	}

	l := ledger.NewDolt(filepath.Join(root, cfg.LedgerDir))
	defer l.Close()

	diff, err := l.Diff()
	if err != nil {
		return err
	}

	if diff == "" {
		fmt.Println("No uncommitted changes in ledger")
	} else {
		fmt.Println(diff)
	}

	return nil
}
