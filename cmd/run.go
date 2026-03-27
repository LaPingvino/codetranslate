package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/LaPingvino/codetranslate/compiler"
	"github.com/LaPingvino/codetranslate/config"
	"github.com/LaPingvino/codetranslate/ledger"
	"github.com/LaPingvino/codetranslate/pipeline"
	"github.com/LaPingvino/codetranslate/translator"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the translation loop",
	Long:  `Translates all pending units: pick next → translate → compile gate → update ledger → repeat.`,
	RunE:  runRun,
}

var (
	flagRunModel string
)

func init() {
	runCmd.Flags().StringVar(&flagRunModel, "model", "", "LLM model to use (overrides config)")
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	root, err := config.ProjectRoot()
	if err != nil {
		return err
	}

	model := cfg.Model
	if flagRunModel != "" {
		model = flagRunModel
	}

	// Set up translator
	t, err := makeTranslator(model, root)
	if err != nil {
		return err
	}

	// Set up compiler
	comp := compiler.ForLanguage(cfg.TargetLang)

	// Open ledger
	ledgerDir := filepath.Join(root, cfg.LedgerDir)
	l := ledger.NewDolt(ledgerDir)

	// Print summary before starting
	summary, err := l.Summary()
	if err != nil {
		return fmt.Errorf("reading ledger: %w", err)
	}
	fmt.Printf("Ledger: %d total, %d todo, %d done, %d failed\n",
		summary.Total, summary.Todo, summary.Done, summary.Failed)

	if summary.Todo == 0 && summary.Failed == 0 {
		fmt.Println("Nothing to translate!")
		return nil
	}

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted — finishing current unit...")
		cancel()
	}()

	p := &pipeline.Pipeline{
		Config:     cfg,
		Ledger:     l,
		Translator: t,
		Compiler:   comp,
	}

	return p.Run(ctx)
}

func makeTranslator(model, root string) (translator.Translator, error) {
	lower := strings.ToLower(model)

	switch {
	case lower == "manual":
		return translator.NewManual(filepath.Join(root, ".codetranslate", "manual")), nil
	case strings.HasPrefix(lower, "ollama:"):
		return translator.NewOllama(strings.TrimPrefix(lower, "ollama:")), nil
	case lower == "ollama":
		return translator.NewOllama(""), nil
	case strings.HasPrefix(lower, "command:"):
		return translator.NewCommand(strings.TrimPrefix(model, "command:")), nil
	default:
		// Assume Claude model
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set (needed for model %q)", model)
		}
		return translator.NewClaude(model), nil
	}
}
