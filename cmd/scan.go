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

	sourceDir := cfg.SourceDir
	if !filepath.IsAbs(sourceDir) {
		sourceDir = filepath.Join(root, sourceDir)
	}
	sourceAbs, err := filepath.Abs(sourceDir)
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

	// Open ledger and batch-add units
	ledgerDir := cfg.LedgerDir
	if !filepath.IsAbs(ledgerDir) {
		ledgerDir = filepath.Join(root, ledgerDir)
	}
	l := ledger.NewDolt(ledgerDir)

	fmt.Printf("Inserting into ledger...")
	added, err := l.AddUnits(units)
	if err != nil {
		return fmt.Errorf("adding units: %w", err)
	}
	fmt.Printf(" done\n")

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
