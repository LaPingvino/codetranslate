package scanner

import "github.com/LaPingvino/codetranslate/ledger"

// Scanner discovers translatable units in a source codebase.
type Scanner interface {
	// Scan walks the source directory and returns all discovered units.
	Scan(sourceDir, sourceLang, targetLang string) ([]*ledger.Unit, error)
}
