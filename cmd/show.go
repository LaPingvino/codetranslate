package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/LaPingvino/codetranslate/config"
	"github.com/LaPingvino/codetranslate/ledger"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <function-name>",
	Short: "Show source, translation, and status for a unit",
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	name := args[0]

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

	// Try as ID first, then search by name
	unit, err := l.GetUnit(name)
	if err != nil {
		// Search by name
		units, err2 := l.ListUnits("")
		if err2 != nil {
			return err2
		}
		for _, u := range units {
			if u.SourceName == name {
				unit = u
				break
			}
		}
		if unit == nil {
			return fmt.Errorf("unit %q not found", name)
		}
	}

	fmt.Printf("ID:      %s\n", unit.ID)
	fmt.Printf("Name:    %s\n", unit.SourceName)
	fmt.Printf("Kind:    %s\n", unit.Kind)
	fmt.Printf("File:    %s → %s\n", unit.SourceFile, unit.TargetFile)
	fmt.Printf("Status:  %s\n", unit.Status)
	fmt.Printf("Tier:    %d\n", unit.Tier)
	fmt.Printf("Model:   %s\n", unit.Model)
	fmt.Printf("Attempts:%d\n", unit.Attempts)

	if unit.LastError != "" {
		fmt.Printf("\nLast Error:\n%s\n", unit.LastError)
	}

	if unit.SourceCode != "" {
		fmt.Printf("\n── Source (%s) ──\n%s\n", cfg.SourceLang, unit.SourceCode)
	}

	if unit.Translation != "" {
		fmt.Printf("\n── Translation (%s) ──\n%s\n", cfg.TargetLang, unit.Translation)
	}

	return nil
}
