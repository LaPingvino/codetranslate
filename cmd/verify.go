package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/LaPingvino/codetranslate/compiler"
	"github.com/LaPingvino/codetranslate/config"
	"github.com/LaPingvino/codetranslate/ledger"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Compile all translations and update ledger status",
	RunE:  runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

func runVerify(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	root, err := config.ProjectRoot()
	if err != nil {
		return err
	}

	comp := compiler.ForLanguage(cfg.TargetLang)
	targetAbs, _ := filepath.Abs(filepath.Join(root, cfg.TargetDir))

	fmt.Printf("Compiling %s code in %s...\n", cfg.TargetLang, targetAbs)

	err = comp.Compile(context.Background(), targetAbs)
	if err != nil {
		fmt.Printf("FAIL: %s\n", err)
	} else {
		fmt.Println("OK: compilation succeeded")
	}

	// Update ledger: mark translated units as compiles (or failed)
	l := ledger.NewDolt(filepath.Join(root, cfg.LedgerDir))
	defer l.Close()

	units, err := l.ListUnits(ledger.StatusTranslated)
	if err != nil {
		return err
	}

	if err == nil {
		// Compilation succeeded — mark all translated as compiles
		for _, u := range units {
			u.Status = ledger.StatusCompiles
			_ = l.UpdateUnit(u)
		}
		if len(units) > 0 {
			_ = l.Commit(fmt.Sprintf("verify: %d units compile", len(units)))
			fmt.Printf("Updated %d units to 'compiles' status\n", len(units))
		}
	}

	return nil
}
