package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/LaPingvino/codetranslate/config"
	"github.com/LaPingvino/codetranslate/ledger"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a translation project",
	Long:  `Sets up the config file and Dolt-backed translation ledger.`,
	RunE:  runInit,
}

var (
	flagSourceDir  string
	flagFromLang   string
	flagTargetDir  string
	flagToLang     string
	flagModel      string
)

func init() {
	initCmd.Flags().StringVar(&flagSourceDir, "source", "", "source code directory")
	initCmd.Flags().StringVar(&flagFromLang, "from", "", "source language (e.g. c++, c, go)")
	initCmd.Flags().StringVar(&flagTargetDir, "target", "", "target code directory")
	initCmd.Flags().StringVar(&flagToLang, "to", "", "target language (e.g. go, c)")
	initCmd.Flags().StringVar(&flagModel, "model", "haiku", "default LLM model")
	initCmd.MarkFlagRequired("source")
	initCmd.MarkFlagRequired("from")
	initCmd.MarkFlagRequired("target")
	initCmd.MarkFlagRequired("to")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()

	// Resolve paths
	sourceAbs, err := filepath.Abs(flagSourceDir)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(flagTargetDir)
	if err != nil {
		return err
	}

	// Check source exists
	if _, err := os.Stat(sourceAbs); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", sourceAbs)
	}

	// Create target dir if needed
	if err := os.MkdirAll(targetAbs, 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	cfg := config.DefaultConfig()
	cfg.SourceDir = flagSourceDir
	cfg.SourceLang = flagFromLang
	cfg.TargetDir = flagTargetDir
	cfg.TargetLang = flagToLang
	cfg.Model = flagModel

	// Save config
	if err := cfg.Save(cwd); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("Created %s\n", config.ConfigFile)

	// Initialize ledger
	ledgerDir := filepath.Join(cwd, cfg.LedgerDir)
	l := ledger.NewDolt(ledgerDir)
	if err := l.Init(); err != nil {
		return fmt.Errorf("initializing ledger: %w", err)
	}
	fmt.Printf("Initialized Dolt ledger at %s\n", cfg.LedgerDir)

	fmt.Printf("\nReady! Next steps:\n")
	fmt.Printf("  translate scan     # discover functions to translate\n")
	fmt.Printf("  translate run      # start translating\n")
	return nil
}
