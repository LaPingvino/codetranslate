package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LaPingvino/codetranslate/compiler"
	"github.com/LaPingvino/codetranslate/config"
	"github.com/LaPingvino/codetranslate/ledger"
	"github.com/LaPingvino/codetranslate/translator"
)

type Pipeline struct {
	Config     *config.Config
	Ledger     ledger.Ledger
	Translator translator.Translator
	Compiler   compiler.Compiler
}

// Run executes the main translation loop until the ledger is empty or an error occurs.
func (p *Pipeline) Run(ctx context.Context) error {
	translated := 0
	failed := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		unit, err := p.Ledger.NextUnit()
		if err != nil {
			return fmt.Errorf("getting next unit: %w", err)
		}
		if unit == nil {
			break // all done
		}

		fmt.Printf("[%s] %s:%s (%s, tier %d)\n", unit.Kind, unit.SourceFile, unit.SourceName, unit.Status, unit.Tier)

		err = p.translateUnit(ctx, unit)
		if err != nil {
			fmt.Printf("  FAILED: %s\n", err)
			failed++
		} else {
			fmt.Printf("  OK → %s\n", unit.Status)
			translated++
		}

		// Commit progress after each unit
		_ = p.Ledger.Commit(fmt.Sprintf("translate %s:%s → %s", unit.SourceFile, unit.SourceName, unit.Status))
	}

	fmt.Printf("\nDone: %d translated, %d failed\n", translated, failed)
	return nil
}

func (p *Pipeline) translateUnit(ctx context.Context, unit *ledger.Unit) error {
	// Mark as WIP
	unit.Status = ledger.StatusWIP
	if err := p.Ledger.UpdateUnit(unit); err != nil {
		return err
	}

	// Gather context from already-translated dependencies in the same tier or lower
	depContext := p.gatherContext(unit)

	for attempt := 0; attempt <= p.Config.MaxRetries; attempt++ {
		unit.Attempts = attempt + 1

		req := &translator.Request{
			SourceCode: unit.SourceCode,
			SourceLang: p.Config.SourceLang,
			TargetLang: p.Config.TargetLang,
			Context:    depContext,
			LastError:  unit.LastError,
		}

		resp, err := p.Translator.Translate(ctx, req)
		if err != nil {
			unit.LastError = err.Error()
			unit.Status = ledger.StatusFailed
			_ = p.Ledger.UpdateUnit(unit)
			return fmt.Errorf("translation attempt %d: %w", attempt+1, err)
		}

		unit.Translation = resp.Code
		unit.Model = resp.Model
		unit.Status = ledger.StatusTranslated

		// Write translation to target file
		if err := p.writeTranslation(unit); err != nil {
			unit.LastError = err.Error()
			unit.Status = ledger.StatusFailed
			_ = p.Ledger.UpdateUnit(unit)
			return fmt.Errorf("writing translation: %w", err)
		}

		// Compile gate
		if p.Compiler != nil {
			absTarget, _ := filepath.Abs(p.Config.TargetDir)
			if err := p.Compiler.Compile(ctx, absTarget); err != nil {
				unit.LastError = err.Error()
				if attempt < p.Config.MaxRetries {
					fmt.Printf("  compile failed (attempt %d/%d), retrying...\n", attempt+1, p.Config.MaxRetries+1)
					continue
				}
				unit.Status = ledger.StatusFailed
				_ = p.Ledger.UpdateUnit(unit)
				return fmt.Errorf("compilation failed after %d attempts: %w", attempt+1, err)
			}
			unit.Status = ledger.StatusCompiles
		}

		unit.LastError = ""
		_ = p.Ledger.UpdateUnit(unit)
		return nil
	}

	return fmt.Errorf("exhausted all retries")
}

func (p *Pipeline) gatherContext(unit *ledger.Unit) string {
	// Get all already-translated units with a lower or equal tier
	units, err := p.Ledger.ListUnits("")
	if err != nil {
		return ""
	}

	var context []string
	for _, u := range units {
		if u.ID == unit.ID {
			continue
		}
		if u.Tier <= unit.Tier && u.Translation != "" {
			context = append(context, fmt.Sprintf("// %s (from %s)\n%s", u.TargetName, u.SourceFile, u.Translation))
		}
	}

	// Limit context size to avoid exceeding token limits
	joined := strings.Join(context, "\n\n")
	if len(joined) > 8000 {
		joined = joined[:8000] + "\n// ... (truncated)"
	}
	return joined
}

func (p *Pipeline) writeTranslation(unit *ledger.Unit) error {
	targetPath := filepath.Join(p.Config.TargetDir, unit.TargetFile)
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// If the target file exists, we need to append/merge rather than overwrite.
	// For now, we use a simple approach: one file per source file, append.
	f, err := os.OpenFile(targetPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "\n// Translated from %s:%s\n%s\n", unit.SourceFile, unit.SourceName, unit.Translation)
	return err
}
