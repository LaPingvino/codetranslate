package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/LaPingvino/codetranslate/config"
	"github.com/LaPingvino/codetranslate/ledger"
	"github.com/LaPingvino/codetranslate/scanner"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan source code and build the translation inventory",
	RunE:  runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	root, err := config.ProjectRoot()
	if err != nil {
		return err
	}

	sourceAbs, err := filepath.Abs(filepath.Join(root, cfg.SourceDir))
	if err != nil {
		return err
	}

	fmt.Printf("Scanning %s for %s code...\n", sourceAbs, cfg.SourceLang)

	s := &scanner.CtagsScanner{}
	units, err := s.Scan(sourceAbs, cfg.SourceLang, cfg.TargetLang)
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	fmt.Printf("Found %d translatable units\n", len(units))

	// Open ledger and add units
	ledgerDir := filepath.Join(root, cfg.LedgerDir)
	l := ledger.NewDolt(ledgerDir)

	added := 0
	for _, u := range units {
		if err := l.AddUnit(u); err != nil {
			fmt.Printf("  warning: failed to add %s:%s: %v\n", u.SourceFile, u.SourceName, err)
			continue
		}
		added++
	}

	// Commit the scan
	_ = l.Commit(fmt.Sprintf("scan: added %d units from %s", added, cfg.SourceDir))

	fmt.Printf("Added %d units to ledger\n", added)

	// Show summary by kind
	summary := map[string]int{}
	for _, u := range units {
		summary[u.Kind]++
	}
	fmt.Println("\nBreakdown:")
	for kind, count := range summary {
		fmt.Printf("  %-12s %d\n", kind, count)
	}

	return nil
}
