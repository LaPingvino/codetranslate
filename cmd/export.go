package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/LaPingvino/codetranslate/config"
	"github.com/LaPingvino/codetranslate/ledger"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export the translation ledger to JSON",
	RunE:  runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
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

	units, err := l.ListUnits("")
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(units, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}
