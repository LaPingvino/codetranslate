package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/LaPingvino/codetranslate/compiler"
	"github.com/LaPingvino/codetranslate/config"
	"github.com/LaPingvino/codetranslate/ledger"
	"github.com/LaPingvino/codetranslate/pipeline"
	"github.com/spf13/cobra"
)

var retryCmd = &cobra.Command{
	Use:   "retry",
	Short: "Retry failed translations with a different model or more context",
	RunE:  runRetry,
}

var (
	flagRetryModel  string
	flagRetryFailed bool
)

func init() {
	retryCmd.Flags().StringVar(&flagRetryModel, "model", "", "use a different model for retries")
	retryCmd.Flags().BoolVar(&flagRetryFailed, "failed", false, "retry all failed units")
	rootCmd.AddCommand(retryCmd)
}

func runRetry(cmd *cobra.Command, args []string) error {
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

	// Reset failed units back to todo
	if flagRetryFailed {
		units, err := l.ListUnits(ledger.StatusFailed)
		if err != nil {
			return err
		}
		if len(units) == 0 {
			fmt.Println("No failed units to retry")
			return nil
		}
		for _, u := range units {
			u.Status = ledger.StatusTodo
			u.Attempts = 0
			u.LastError = ""
			if err := l.UpdateUnit(u); err != nil {
				return fmt.Errorf("resetting %s: %w", u.SourceName, err)
			}
		}
		_ = l.Commit(fmt.Sprintf("retry: reset %d failed units to todo", len(units)))
		fmt.Printf("Reset %d failed units to todo\n", len(units))
	}

	// Now run the pipeline
	model := cfg.Model
	if flagRetryModel != "" {
		model = flagRetryModel
	}

	t, err := makeTranslator(model, root)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		cancel()
	}()

	p := &pipeline.Pipeline{
		Config:     cfg,
		Ledger:     l,
		Translator: t,
		Compiler:   compiler.ForLanguage(cfg.TargetLang),
	}

	return p.Run(ctx)
}
