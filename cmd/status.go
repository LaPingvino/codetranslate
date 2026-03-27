package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

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

	ledgerDir := cfg.LedgerDir
	if !filepath.IsAbs(ledgerDir) {
		ledgerDir = filepath.Join(root, ledgerDir)
	}
	l := ledger.NewDolt(ledgerDir)
	defer l.Close()

	s, err := l.Summary()
	if err != nil {
		return err
	}

	// Header
	fmt.Println()
	fmt.Println("  ┌─────────────────────────────────────────────────┐")
	fmt.Printf("  │  codetranslate  %s -> %s%s│\n",
		cfg.SourceLang, cfg.TargetLang,
		pad(50-19-len(cfg.SourceLang)-len(cfg.TargetLang)-4))
	fmt.Println("  ├─────────────────────────────────────────────────┤")

	// Progress bar
	if s.Total > 0 {
		pct := float64(s.Done+s.Compiles+s.Tested) / float64(s.Total) * 100
		bar := colorBar(s, 40)
		fmt.Printf("  │  %s  %3.0f%%  │\n", bar, pct)
	} else {
		fmt.Printf("  │  %-47s│\n", "no units scanned yet")
	}

	fmt.Println("  ├─────────────────────────────────────────────────┤")

	// Status breakdown with inline bar charts
	type row struct {
		label string
		count int
		color string
		ch    byte
	}
	rows := []row{
		{"done", s.Done + s.Compiles + s.Tested, "\033[32m", '#'},
		{"translated", s.Translated, "\033[36m", '>'},
		{"wip", s.WIP, "\033[33m", '~'},
		{"todo", s.Todo, "\033[37m", '.'},
		{"failed", s.Failed, "\033[31m", '!'},
	}

	for _, r := range rows {
		w := 0
		if s.Total > 0 {
			w = r.count * 20 / s.Total
		}
		filled := strings.Repeat(string(r.ch), w)
		empty := strings.Repeat(" ", 20-w)
		fmt.Printf("  │  %-12s %s%s\033[0m%s %5d  │\n", r.label, r.color, filled, empty, r.count)
	}

	fmt.Println("  ├─────────────────────────────────────────────────┤")
	fmt.Printf("  │  total%41d  │\n", s.Total)
	fmt.Println("  └─────────────────────────────────────────────────┘")

	// Source/target info
	fmt.Printf("\n  source: %s\n  target: %s\n\n", cfg.SourceDir, cfg.TargetDir)

	return nil
}

func colorBar(s *ledger.Summary, width int) string {
	done := (s.Done + s.Compiles + s.Tested) * width / s.Total
	trans := s.Translated * width / s.Total
	wip := s.WIP * width / s.Total
	failed := s.Failed * width / s.Total
	todo := width - done - trans - wip - failed

	bar := "\033[32m" + strings.Repeat("█", done)   // green
	bar += "\033[36m" + strings.Repeat("▓", trans)   // cyan
	bar += "\033[33m" + strings.Repeat("▒", wip)     // yellow
	bar += "\033[31m" + strings.Repeat("░", failed)  // red
	bar += "\033[90m" + strings.Repeat("░", todo)    // gray
	bar += "\033[0m"
	return bar
}

func pad(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}
