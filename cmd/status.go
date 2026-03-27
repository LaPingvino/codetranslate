package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/LaPingvino/codetranslate/config"
	"github.com/LaPingvino/codetranslate/ledger"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show translation progress summary",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	s, err := l.Summary()
	if err != nil {
		return err
	}

	fmt.Printf("Translation Status: %s → %s\n", cfg.SourceLang, cfg.TargetLang)
	fmt.Printf("Source: %s → Target: %s\n\n", cfg.SourceDir, cfg.TargetDir)

	bar := progressBar(s)
	fmt.Println(bar)
	fmt.Println()

	fmt.Printf("  todo         %d\n", s.Todo)
	fmt.Printf("  wip          %d\n", s.WIP)
	fmt.Printf("  translated   %d\n", s.Translated)
	fmt.Printf("  compiles     %d\n", s.Compiles)
	fmt.Printf("  tested       %d\n", s.Tested)
	fmt.Printf("  done         %d\n", s.Done)
	fmt.Printf("  failed       %d\n", s.Failed)
	fmt.Printf("  ─────────────\n")
	fmt.Printf("  total        %d\n", s.Total)

	if s.Total > 0 {
		pct := float64(s.Done+s.Compiles+s.Tested) / float64(s.Total) * 100
		fmt.Printf("\n  %.0f%% complete\n", pct)
	}

	return nil
}

func progressBar(s *ledger.Summary) string {
	if s.Total == 0 {
		return "[no units]"
	}
	width := 40
	done := (s.Done + s.Compiles + s.Tested) * width / s.Total
	wip := (s.WIP + s.Translated) * width / s.Total
	failed := s.Failed * width / s.Total
	todo := width - done - wip - failed

	bar := "["
	for i := 0; i < done; i++ {
		bar += "="
	}
	for i := 0; i < wip; i++ {
		bar += ">"
	}
	for i := 0; i < failed; i++ {
		bar += "!"
	}
	for i := 0; i < todo; i++ {
		bar += " "
	}
	bar += "]"
	return bar
}
